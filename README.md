# ansible-puller

This daemon extends the `ansible-pull` method of running Ansible.
It uses S3 or HTTP file transmission instead of Git to manage distribution (easy to cache), and integrates with Prometheus monitoring.

## Why ansible-puller?

`ansible-pull` assumes that you are checking out an Ansible repository from git.
This wasn't an option for us at the scale that we needed, so we turned to HTTP file distribution.
On top of scaling, we've integrated monitoring (via Prometheus) to retain the centralized view of all of our Ansible
runs and a simple REST API to enable/disable the puller and trigger a run to give more fine-grained control of rollouts.

# How to use it

Ansible puller expects an HTTP endpoint, or an S3 ARN that points to a tarball with Ansible playbooks, inventories, etc.

The minimal configuration would just be a config file supplying `http-url` (see below).
While the defaults have been set assuming [Ansible's "Alternative Directory Layout"](https://docs.ansible.com/ansible/latest/user_guide/playbooks_best_practices.html#alternative-directory-layout)
it should be configurable enough to support alternative setups.

Inside of the tarball, at a minimum you'll need an inventory, playbook, and a `requirements.txt` file.
The requirements file will be used to populate the Python virtual environment that Ansible will run in locally.
At a minimum it needs to contain `ansible` so that Ansible will be installed in the virtualenv. A pinned version is even better.
The playbook is what will be actually run.
The inventory needs to contain the hostname of the node that Ansible puller is installed on.

## Ansible Inventory

To support our use of an Infrastructure monorepo, Ansible-puller will loop through an entire directory looking for inventories.
It will test each of these inventories for a matching hostname and run the given playbook in the first found inventory.

Given the structure:

```
.
├── ansible.cfg
├── inventories/
│   ├── production/
│   └── staging/
├── roles/
└── site.yml
```

And the config file:

```json
{
  "http-url": "https://example.com/infra.tgz",
  "ansible-inventory": ["inventories/production", "inventories/staging"],
  "playbook": "site.yml"
}
```

Setting `ansible-inventory` to `["inventories/production", "inventories/staging"]` and `playbook` to `site.yml`
would mean that the puller would search for the correct host in `production` and `staging`, provided that the hosts were
a part of `site.yml`'s run. Use the `debug` option to get more insight to the process while it is running.

## Configuration and Metrics

Config file should be in: `/etc/ansible-puller/config.json`, `$HOME/.ansible-puller.json`, `./ansible-puller.json`

| Config Option            | Default                               | Description                                                                             |
|--------------------------|---------------------------------------|-----------------------------------------------------------------------------------------|
| `http-listen-string`     | `"0.0.0.0:31836"`                     | Address/port the service will listen on. Use `127.0.0.1:31386` to lock down the UI.     |
| `http-proto`             | `https`                               | Modify to "http" if necessary                                                           |
| `http-user`              | `""`                                  | Username for HTTP Basic Auth                                                            |
| `http-pass`              | `""`                                  | Password for HTTP basic Auth                                                            |
| `http-url`               | `""`                                  | HTTP Url to find the Ansible tarball. Required if s3-arn is not set                     |
| `http-checksum-url`      | `""`                                  | HTTP Url to find the Ansible tarball md5 hash. Defaults to http-url + `.md5`.           |
| `log-dir`                | `"/var/log/ansible-puller"`           | Log directory (must exist)                                                              |
| `ansible-dir`            | `""`                                  | Path in the pulled tarball to cd into before ansible commands - usually ansible.cfg dir |
| `ansible-playbook`       | `"site.yml"`                          | The playbook that will be run  - relative to ansible-dir                                |
| `ansible-inventory`      | `[]`                                  | List of inventories to operate on - relative to ansible-dir                             |
| `venv-python`            | `"/usr/bin/python3"`                  | Path to the python version you are using for Ansible                                    |
| `venv-path`              | `"/root/.virtualenvs/ansible_puller"` | Path to where the virtualenv will be created                                            |
| `venv-requirements-file` | `"requirements.txt"`                  | Path to the python requirements file to populate the virtual environment                |
| `sleep`                  | `30`                                  | How often to trigger run events in minutes                                              |
| `start-disabled`         | `false`                               | Whether or not to start with Ansbile disabled (good for debugging)                      |
| `s3-arn`                 | `""`                                  | S3 location to find the Ansible tarball. Required if http-url is not set                |
| `s3-conn-region`         | `""`                                  | S3 connection region to use. Uses the aws-sdk-go-v2 default providers if not set        |
| `debug`                  | `false`                               | Whether or not to start in debug mode                                                   |
| `once`                   | `false`                               | Only run the configured playbook once and then stop                                     |

### Monitoring with prometheus

This daemon uses Ansible's `json` STDOUT callback to parse the results of this run for this host.
It currently produces the number of tasks that are ok, skipped, changed, failed, or unreachable.

| Metric                            | Description                                                  |
|-----------------------------------|--------------------------------------------------------------|
| `ansible_puller_debug`            | Whether or not debug mode is enabled                         |
| `ansible_puller_disabled`         | Whether or not the puller is disabled                        |
| `ansible_puller_last_success`     | Last timestamp of a successful run                           |
| `ansible_puller_last_exit_code`   | Last ansible run exit code                                   |
| `ansible_puller_play_summary`     | Ansible metrics: changed, failures, ok, skipped, unreachable |
| `ansible_puller_run_time_seconds` | How long Ansible took to run to completion                   |
| `ansible_puller_running`          | Whether or not the puller is currently running               |
| `ansible_puller_runs`             | How many times the puller has run                            |
| `ansible_puller_version`          | Version (git sha) of the puller                              |

#### Exit code mapping

Following are some of internal ansible-puller errors and corresponding exit codes 

| Exit code  | Error explained               | 
|------------|-------------------------------|
| 2          | virtualenv creation failed    |
| 5          | virtual env update failed     |
| 6          | inventory fetch failed        |
|125         | Remote repository pull failed |


### MD5 checksum support

Enabling MD5 checksumming will prevent extraneous calls to download the ansible tarball from the
remote.

By design, ansible_puller will look at the remote path `<resource_path>.md5` to discover the live
MD5 checksum. If, for example, your resource is located at `https://example.com/some/file.tgz` then
ansible_puller will look for the MD5 hash at `https://example.com/some/file.tgz.md5`. A custom remote
path can be specified with the `http-checksum-url` option.
The following conditions will lead to a (re-)download of the ansible tarball:
- There is no current ansible tarball at the specified local path
- The current hash of the local ansible tarball not match the remote checksum
- The remote checksum does not exist

If a remote checksum exists then the downloaded tarball will be hashed and the resulting output will
be compared to the remote checksum to validate artifact integrity.

## Runtime Dependencies

This program expects the following to be true about its runtime environment:
* It is running as root (unless you don't need `--become`)
* `virtualenv` is installed on the server

## Development Notes

This project uses Go Modules. Go 1.19+ should be able to handle this transparently.

### Doing Things

#### Running Locally

`bazelisk run //:ansible_puller`

or, without bazel

`go run .`

#### Running tests

`bazelisk test //...`


#### Building a Production Release

`bazelisk build --config=release --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //...`

#### Building Production Packages

* `bazelisk build --config=release //:ansible_puller_deb`
* `bazelisk build --config=release //:ansible_puller_rpm`


#### Debugging an Ansible Run

For debugging the application, use the `--debug` flag, or the `debug` option in the config file.
This streams the Ansible output to the console so that you can follow along in the run.

Also consider using the `--once` flag to run the process just once and then exit without spinning up the webserver.
