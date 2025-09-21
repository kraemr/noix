# WARNING
This project is still very early in development and can under certain circumstances crash your system or destroy data!! 
You have been warned.

# noix

Tired of overly complex "reproducible" systems, well this repo aims to create a human friendly reproduction system through use of chroots, with some magic for kernel params or switching kernels

This functions similiarly to docker, but with less complexity

# What is this for?
With noix you can create reproducible chroot environments to function as dev environments or for shipping kernel with specific patches params etc.

# Why use this instead of Docker,Nix or other "reproducibility programs"

This approach has an almost non-existant performance impact and gives you the user the maximum amount of control. 

Docker has the problem of greatly decreasing performance, making it unusable for certain tasks, like realtime application.

NixOs and Nix in particular work well for realtime applications for example (kernel config, kernel params, kernel modules, version management) etc. BUT the problem with Nix is the complexity of achieving this.

Apparently simply the nix glue code for nixpkgs is about 2.4 million lines of code ...

Docker also hogs all your ressources, since every dockerbuild is self contained and contains an "OS" or atleast the structure.

If you like you can do the same with this software.
However it is more intended to supply the minimum amount required for a certain application or usecase
For example this config makes bash run in a chroot

```toml
root = "/etc/noix/"
name = "minimal_bash"
bootable = false
immutable = false

bind_mounts = [
 "/proc",
]

create_dirs = [
 "/tmp",
]

sym_links = [
    ["/lib","usr/lib"],
    ["/bin","usr/bin"],
]

sync_paths = [
 "/usr/bin/bash",
 "/usr/lib/x86_64-linux-gnu/libtinfo.so.6",
 "/lib/x86_64-linux-gnu/libc.so.6",
 "/lib/x86_64-linux-gnu/libselinux.so.1",
 "/lib/x86_64-linux-gnu/libcap.so.2",
 "/lib/x86_64-linux-gnu/libpcre2-8.so.0",
 "/lib/x86_64-linux-gnu/libselinux.so.1",
 "/lib64",
]
```