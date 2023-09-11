# CPUSH

A go language tool to work with SSH to cisco devices.

It allows to collect the output of commands directly, without logging in. It can cache passwords.

# To install

    go install github.com/cdevr/cpush@latest

# Example execution

    cpush --device ip-rtr-ch-1 --cmd 'show version'

You can also pass parameters just directly if just specifying device and command:

    cpush ip-rtr-ch-1 show version

In this case, cpush will interpret the first argument as the router name, and subsequent arguments will be used to send a command.

# Run commands on many devices

List a bunch of devices in a file called "devices_shver". For example router1 and router2

    # cat devices_shver
    router1
    router2
    # cpush --devicefile devices_shver --cmd "show version" --logOutputTemplate "shver_%s"
    # ls
    devices_shver
    shver_router1
    shver_router2
    #

This will create the files "shver_router1" and "shver_router2" that contain the output of the listed devices.