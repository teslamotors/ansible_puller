package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	appName               = "ansible-puller"
	hostname              = ""
	ansibleDisabled       = false
	ansibleRunning        = false
	ansibleLastRunSuccess = true
	Version               string

	// Prometheus Metrics
	promAnsibleIsRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ansible_puller_running",
		Help: "Whether or not Ansible-Pull is currently running",
	})
	promAnsibleRuns = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "ansible_puller_runs",
		Help: "Number of Ansible-Pull runs",
	})
	promAnsibleRunTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ansible_puller_run_time_seconds",
		Help: "Time it took ansible to run",
	})
	promAnsibleIsDisabled = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ansible_puller_disabled",
		Help: "Whether or not Ansible-Pull is currently locked/disabled",
	})
	promAnsibleLastSuccess = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ansible_puller_last_success",
		Help: "UTC Epoch timestamp of last Successful Ansible run",
	})
	promAnsibleSummary = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ansible_puller_play_summary",
		Help: "Play status for Ansible run",
	},
		[]string{"status"},
	)
	promVersion = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ansible_puller_version",
		Help: "Current running version of Ansible Puller",
	},
		[]string{"version"},
	)
	promDebug = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ansible_puller_debug",
		Help: "Whether or not Ansible Puller is running in debug mode",
	})
)

func init() {
	prometheus.MustRegister(promAnsibleIsRunning)
	prometheus.MustRegister(promAnsibleIsDisabled)
	prometheus.MustRegister(promAnsibleRuns)
	prometheus.MustRegister(promAnsibleRunTime)
	prometheus.MustRegister(promAnsibleLastSuccess)
	prometheus.MustRegister(promAnsibleSummary)
	prometheus.MustRegister(promVersion)
	prometheus.MustRegister(promDebug)

	viper.SetConfigName(appName)
	viper.AddConfigPath(fmt.Sprintf("/etc/%s/", appName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s", appName))
	viper.AddConfigPath(".")

	pflag.String("http-listen-string", "0.0.0.0:31836", "IP:Port combination the server should listen on")
	pflag.String("http-proto", "https", "Set to 'http' if necessary")
	pflag.String("http-user", "", "HTTP username for pulling the remote file")
	pflag.String("http-pass", "", "HTTP password for pulling the remote file")

	pflag.String("http-url", "", "Remote endpoint to retrieve the file from")
	pflag.String("s3-uri", "", "Remote endpoint in S3 to retrieve the file from if s3-arn isn't used")
	pflag.String("s3-arn", "", "Remote endpoint in S3 to retrieve the file from if s3-uri isn't used")

	pflag.String("log-dir", "/var/log/"+appName, "Logging directory")
	pflag.StringSlice("ansible-inventory", []string{}, "List of ansible inventories to look in, comma-separated, relative to ansible-dir")
	pflag.String("ansible-playbook", "site.yml", "Path in the pulled tarball to the playbook to run, relative to ansible-dir")
	pflag.String("ansible-dir", "", "Path in the pulled tarball to cd into before ansible commands - usually dir where ansible.cfg is")

	pflag.String("venv-python", "/usr/bin/python3", "Path to the Python executable to be used for building the virtual environment")
	pflag.String("venv-path", "/root/.virtualenvs/ansible_puller", "Path to house the virtual environment")
	pflag.String("venv-requirements-file", "requirements.txt", "Relative path in the pulled tarball of the requirements file to populate the virtual environment")

	pflag.Int("sleep", 30, "Number of minutes to sleep between runs")
	pflag.Bool("start-disabled", false, "Whether or not to start the server disabled")
	pflag.Bool("debug", false, "Start the server in debug mode")
	pflag.Bool("once", false, "Run Ansible Puller just once, then exit")

	err := viper.ReadInConfig()
	if err != nil {
		logrus.Fatalf("fatal error in config file: %s", err)
	}

	err = viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		logrus.Fatal("unable to bind configuration")
	}

	pflag.Parse()

	logrus.SetOutput(os.Stdout)
	if viper.GetBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
		promDebug.Set(1)
	} else {
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}

	if viper.GetBool("start-disabled") {
		ansibleDisable()
	}

	hostname, err = os.Hostname()
	if err != nil {
		logrus.Fatal("Unable to detect hostname")
	}

}

func ansibleDisable() {
	ansibleDisabled = true
	promAnsibleIsDisabled.Set(1)
	logrus.Infoln("Disabled Ansible-Puller")
}

func ansibleEnable() {
	ansibleDisabled = false
	promAnsibleIsDisabled.Set(0)
	logrus.Infoln("Enabled Ansible-Puller")
}

func getAnsibleRepository(context context.Context, runDir string) error {
	httpURL := viper.GetString("http-url")
	s3URI := viper.GetString("s3-uri")
	s3ARN := viper.GetString("s3-arn")
	nonEmptyResourceParameters := 0
	for _, resource := range []string{httpURL, s3URI, s3ARN} {
		if resource != "" {
			nonEmptyResourceParameters++
		}
	}
	if nonEmptyResourceParameters != 1 {
		return errors.New("exactly one remote resource must be specified. Choose one 'http-url', 's3-uri', or 's3-arn'")
	}

	localCacheFile := fmt.Sprintf("/tmp/%s.tgz", appName)

	if httpURL != "" {
		remoteHTTPURL := fmt.Sprintf("%s://%s", viper.GetString("http-proto"), httpURL)
		err := idempotentFileDownload(remoteHTTPURL, localCacheFile, viper.GetString("http-user"), viper.GetString("http-pass"))
		if err != nil {
			return errors.Wrap(err, "unable to pull Ansible repo")
		}
	}

	if s3URI != "" {
		bucketObject, err := parseS3ResourceFromURI(s3URI)
		if err != nil {
			return err
		}
		err = downloadS3File(context, bucketObject, localCacheFile)
		if err != nil {
			return errors.Wrap(err, "unable to pull S3 bucket") 
		}
	}

	if s3ARN != "" {
		bucketObject, err := parseS3ResourceFromARN(s3ARN)
		if err != nil {
			return err
		}
		err = downloadS3File(context, bucketObject, localCacheFile)
		if err != nil {
			return errors.Wrap(err, "unable to pull S3 bucket") 
		}
	}

	err := extractTgz(localCacheFile, runDir)
	if err != nil {
		return errors.Wrap(err, "unable to extract tgz")
	}

	return nil
}

// Core run logic
func ansibleRun(context context.Context) error {
	if ansibleDisabled {
		logrus.Infoln("Tried to run Ansible, but currently disabled. Skipping.")
		return nil
	}

	ansibleRunning = true
	promAnsibleIsRunning.Set(1)

	defer func() {
		ansibleRunning = false
		promAnsibleIsRunning.Set(0)
		promAnsibleRuns.Inc()
	}()

	runID := uuid.NewV4().String()
	runLogger := logrus.WithFields(logrus.Fields{"run_id": runID})

	runLogger.Infoln("Creating tmpdir for execution")
	runDir, err := ioutil.TempDir("", appName)
	if err != nil {
		logrus.Fatal(err)
	}
	if !viper.GetBool("debug") {
		defer os.RemoveAll(runDir)
	}

	runLogger.Infoln("Pulling remote repository")
	if err = getAnsibleRepository(context, runDir); err != nil {
		runLogger.Errorln("Unable to pull ansible repository: ", err)
		return err
	}

	vCfg := VenvConfig{
		Path:   viper.GetString("venv-path"),
		Python: viper.GetString("venv-python"),
	}

	runLogger.Infoln("Ensuring virtualenv exists")
	if err = vCfg.Ensure(); err != nil {
		return err
	}
	runLogger.Infoln("Updating virtualenv")
	if err = vCfg.Update(filepath.Join(runDir, viper.GetString("venv-requirements-file"))); err != nil {
		return err
	}

	aCfg := AnsibleConfig{
		VenvConfig:    vCfg,
		Cwd:           filepath.Join(runDir, viper.GetString("ansible-dir")),
		InventoryList: viper.GetStringSlice("ansible-inventory"),
	}

	runLogger.Infoln("Finding inventory for the current host")
	inventory, target, err := aCfg.FindInventoryForHost()
	if err != nil {
		return err
	}

	ansibleRunner := AnsiblePlaybookRunner{
		AnsibleConfig:   aCfg,
		PlaybookPath:    viper.GetString("ansible-playbook"),
		InventoryPath:   inventory,
		LimitExpr:       target,
		LocalConnection: true,
	}

	runLogger.Infoln("Starting Ansible run")

	runOutput, ansibleRunErr := ansibleRunner.Run()
	if ansibleRunErr == nil {
		promAnsibleLastSuccess.Set(float64(time.Now().Unix()))
	}

	promAnsibleSummary.WithLabelValues("ok").Set(float64(runOutput.Stats[hostname].Ok))
	promAnsibleSummary.WithLabelValues("skipped").Set(float64(runOutput.Stats[hostname].Skipped))
	promAnsibleSummary.WithLabelValues("changed").Set(float64(runOutput.Stats[hostname].Changed))
	promAnsibleSummary.WithLabelValues("failures").Set(float64(runOutput.Stats[hostname].Failures))
	promAnsibleSummary.WithLabelValues("unreachable").Set(float64(runOutput.Stats[hostname].Unreachable))

	runLogger.Infoln("Writing ansible output to logfile")

	err = ioutil.WriteFile(viper.GetString("log-dir")+"/ansible-run-output.log", []byte(runOutput.Stdout), 0600)
	if err != nil {
		runLogger.Errorln("Unable to write Ansible output to log file: ", err)
	}

	err = ioutil.WriteFile(viper.GetString("log-dir")+"/ansible-run-error.log", []byte(runOutput.Stderr), 0600)
	if err != nil {
		runLogger.Errorln("Unable to write Ansible output to log file: ", err)
	}

	runLogger.Infoln("All done, going to sleep")
	return ansibleRunErr
}

func main() {
	context := context.Background()

	if viper.GetBool("once") {
		if err := ansibleRun(context); err != nil {
			logrus.Fatalln("Ansible run failed due to: " + err.Error())
		}

		return
	}

	promVersion.WithLabelValues(Version).Set(1)

	period := time.Duration(viper.GetInt("sleep")) * time.Minute
	tickerChan := time.NewTicker(period).C
	runChan := make(chan bool, 8)

	go func() {
		// Blocking wait for the timer to tick, then send a notification down the run channel
		// This will tie the timer and ad-hoc jobs to the same channel so that we can simplify run triggers
		for {
			<-tickerChan
			runChan <- true
		}
	}()

	go func() {
		logrus.Infoln(fmt.Sprintf("Launching Ansible Runner. Runs %d minutes apart.", viper.GetInt("sleep")))
		runChan <- true
		for {
			<-runChan

			start := time.Now()
			err := ansibleRun(context)
			elapsed := time.Since(start)

			promAnsibleRunTime.Set(elapsed.Seconds())

			if err != nil {
				logrus.Errorln("Ansible run failed due to: " + err.Error())
				ansibleLastRunSuccess = false
			} else {
				ansibleLastRunSuccess = true
			}
		}
	}()

	srv := NewServer(runChan)
	logrus.Infoln("Starting server on " + viper.GetString("http-listen-string"))
	logrus.Fatal(srv.ListenAndServe())
}
