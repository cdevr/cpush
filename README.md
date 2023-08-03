# CPUSH

A go language tool to work with SSH to cisco devices.

It allows to collect the output of commands directly, without logging in. It can cache passwords.

# To install

    go install github.com/cdevr/cpush@latest

# Example execution

    cpush --device ip-rtr-ch-1 --cmd 'show version'
