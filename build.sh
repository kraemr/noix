#!/bin/sh
cd src/
go build
cd ..
mv src/noix noix
#setcap cap_sys_chroot+ep
