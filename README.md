**Welcome to CPUSH: Go Language Tool for managing network devices using SSH**

CPUSH is an easy-to-use command-line tool that allows you to interact with Cisco devices via SSH, without needing to log in. It also comes with password caching capabilities.

**Getting Started**

To install CPUSH, simply run the following command:

```bash
go install github.com/cdevr/cpush/cmd/cpush@latest
```

**Using CPUSH for the First Time**

Here are a few examples of how you can use CPUSH to retrieve output from Cisco devices:

* Run a specific command on a device: `cpush --device ip-rtr-ch-1 --cmd 'show version'`
* Pass parameters directly (e.g., specify just the device and command): `cpush ip-rtr-ch-1 show version`
* Enable an interactive session with a router: `cpush --device ip-rtr-ch-1 -i`

**Shortcuts**

If you're familiar with CPUSH, you can use shortcuts to simplify your workflow:

* Run a single command with all the necessary parameters: `cpush ip-rtr-ch-1 sh ver`
* Use the `-i` flag for interactive sessions: `cpush -i ip-rtr-ch-1`

**Running Multiple Commands on Multiple Devices**

You can list multiple devices in a file called "devices_shver" and run commands on each one. Here's an example:

```bash
# cat devices_shver
router1
router2
# cpush --device file:devices_shver --cmd "show version" --output "shver_%s"
```

This will create two files, `shver_router1` and `shver_router2`, containing the output from each device. This
will work fine, even if there's thousands of devices in the device list file.

**Configuring Devices**

CPUSH has special logic to apply configuration changes "atomically" (almost atomically). The --push flag.
Here's an example:

```bash
# cpush --device ip-rtr-1 --push 'int lo 99; ip addr 1.0.0.1 255.255.255.0'
```

**Config file for cpush itself**

You can put default options for cpush in a file called `~/.cpush`, for example specifying a proxy server. For example:

```
socks: 81.83.85.87:3333
```

You can put a default value for any flag you want. So just use cpush --help to find out what options are supported.

**Developing on This Repository**

To take full advantage of this repository, please configure your Git hooks by running the following command:

```bash
git config core.hooksPath .githooks
```

This will enable the hooks to run automatically when you push changes to the repository.
