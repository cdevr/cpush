# CPUSH

A go language tool to work with SSH to cisco devices.

It allows to collect the output of commands directly, without logging in. It can cache passwords.

# To install

    go install github.com/cdevr/cpush@latest

# To just run directly

    go run github.com/cdevr/cpush

# Example execution

    cpush --device ip-rtr-ch-1 --cmd 'show version'
    go run github.com/cdevr/cpush --device ip-sw-ch-fra-5 --cmd 'show ip int brief'
