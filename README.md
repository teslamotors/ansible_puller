# ansible-puller

This daemon extends the `ansible-pull` method of running Ansible.
It uses HTTP file transmission instead of Git to manage distribution (easy to cache), and integrates with Prometheus monitoring.

## Why ansible-puller?

`ansible-pull` assumes that you are checking out an Ansible repository from git.
This wasn't an option for us at the scale that we needed, so we turned to HTTP file distribution.
On top of scaling, we've integrated monitoring (via Prometheus) to retain the centralized view of all of our Ansible 
runs and a simple REST API to enable/disable the puller and trigger a run to give more fine-grained control of rollouts.

# How to use it

Ansible puller expects an HTTP endpoint with a tarball full of Ansible playbooks, inventories, etc.

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

| Config Option            | Default                               | Description                                                                             | Required |
|--------------------------|---------------------------------------|-----------------------------------------------------------------------------------------|----------|
| `http-listen-string`     | `"0.0.0.0:31836"`                     | Address/port the service will listen on. Use `127.0.0.1:31386` to lock down the UI.     |          |
| `http-proto`             | `https`                               | Modify to "http" if necessary                                                           |          |
| `http-user`              | `""`                                  | Username for HTTP Basic Auth                                                            |          |
| `http-pass`              | `""`                                  | Password for HTTP basic Auth                                                            |          |
| `http-url`               | `""`                                  | HTTP Url to find the Ansible tarball                                                    | yes      |
| `log-dir`                | `"/var/log/ansible-puller"`           | Log directory (must exist)                                                              |          |
| `ansible-dir`            | `""`                                  | Path in the pulled tarball to cd into before ansible commands - usually ansible.cfg dir |          |
| `ansible-playbook`       | `"site.yml"`                          | The playbook that will be run  - relative to ansible-dir                                |          |
| `ansible-inventory`      | `[]`                                  | List of inventories to operate on - relative to ansible-dir                             |          |
| `venv-python`            | `"/usr/bin/python3"`                  | Path to the python version you are using for Ansible                                    |          |
| `venv-path`              | `"/root/.virtualenvs/ansible_puller"` | Path to where the virtualenv will be created                                            |          |
| `venv-requirements-file` | `"requirements.txt"`                  | Path to the python requirements file to populate the virtual environment                |          |
| `sleep`                  | `30`                                  | How often to trigger run events in minutes                                              |          |
| `start-disabled`         | `false`                               | Whether or not to start wth Ansbile disabled (good for debugging)                       |          |
| `debug`                  | `false`                               | Whether or not to start in debug mode                                                   |          |

### Monitoring with prometheus

This daemon uses Ansible's `json` STDOUT callback to parse the results of this run for this host.
It currently produces the number of tasks that are ok, skipped, changed, failed, or unreachable.


| Metric                        | Description                                             |
|-------------------------------|---------------------------------------------------------|
| `ansible_puller_runs`         | The number of runs since the daemon started             | 
| `ansible_puller_running`      | Whether or not the puller is currently preforming a run |
| `ansible_puller_disabled`     | Whether or not the puller is disabled                   | 
| `ansible_puller_last_success` | Timestamp of the last successful run                    | 
| `ansible_play_summary`        | List of task statuses for the host                      | 

## Runtime Dependencies

This program expects the following to be true about its runtime environment:
* It is running as root (unless you don't need `--become`)
* `virtualenv` is installed on the server

## Development Notes

This project uses Go Modules. Go 1.11+ should be able to handle this transparently.

[packr](https://github.com/gobuffalo/packr) is used as a wrapper to build the HTML files directly into the Go binary.
Anywhere you would use the `go` command you can substitute `packr`. For development, use the normal `go run .`,
as packr is smart enough to read from disk when the strings are not found in memory.

Note that this project uses packer v2.

### Doing Things

To run locally: `go run .`

To build a production release: `GOOS=linux GOARCH=amd64 packr2 build` (see [build-release.sh](build-release.sh))

For debugging the application, use the `--debug` flag, or the `debug` option in the config file. It'll make the logs a little easier to read.
