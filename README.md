## hide

Basic forward-proxy written in Go.

#### Usage

Deploy a forward proxy, hide your IP address.

> [!IMPORTANT]  
> AWS is the only "deployer/provider" option available for now, so you need an AWS account with credentials set in your environment.

Simple:
- `./hide deploy aws --ipAddr <YourIp>`

Custom domain:
- `./hide deploy aws --ipAddr <YourIp> --hostedZone custom.domain`

