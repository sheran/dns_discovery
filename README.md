## DNS Discovery

** This is WIP **

Need to use this in conjuction with my [dnsscan](https://github.com/sheran/dnsscan) docker image. The image is a self-contained gobuster/unbound package that will do a DNS guessing/bruteforcing attack.

Once you've built the dnsscan image, then you can run this binary. It takes one argument which is the TLD that you want to scan and then executes the DNS guessing. Once the DNS scan is done, it will do a hostname to ip conversion and output the IP and hosts that are on it in json format.
