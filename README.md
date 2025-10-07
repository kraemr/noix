# WARNING
This project is still very early in development and can under certain circumstances crash your system or destroy data!! 
You have been warned.

# noix

Tired of overly complex "reproducible" systems, well this repo aims to create a human friendly reproduction system through use of chroots, with some magic for kernel params or switching kernels

This functions similiarly to docker, but with less complexity

# What is this for?
With noix you can create reproducible chroot environments to function as dev environments or for shipping kernel with specific patches params etc.



# Quick Example

```bash
# 1. Copies files to "container" binds mounts, creates symlinks etc
./noix -c=build -config=examples/alpine.toml
# 2. Run a program in the "container"
./noix -c=run -chroot=/tmp/alpine/1234 -exec=sh
```


# Why use this instead of Docker,Nix or other "reproducibility programs"

This approach has an almost non-existant performance impact and gives you the user the maximum amount of control. 

Docker has the problem of greatly decreasing performance, making it unusable for certain tasks, like realtime application.

NixOs and Nix in particular work well for realtime applications for example (kernel config, kernel params, kernel modules, version management) etc. BUT the problem with Nix is the complexity of achieving this.

Apparently simply the nix glue code for nixpkgs is about 2.4 million lines of code ...

Docker also hogs all your ressources, since every dockerbuild is self contained and contains an "OS" or atleast the structure.

If you like you can do the same with this software.
However it is more intended to supply the minimum amount required for a certain application or usecase
For example this config makes dash run in a chroot with networking

```toml
root = "/tmp/"
name = "alpine1234"
base = [
    "./alpine-minirootfs-3.22.1-x86_64.tar.gz"
]
# This is needed for networking
sync_paths = [
    "/etc/resolv.conf"
]

```