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
