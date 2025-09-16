

echo $1
echo $2

input_path=$1
chroot_path=$2
strippedOldPath=${input_path#/}
newPath=$chroot_path$strippedOldPath
echo $newPath

rsync -a $input_path $newPath
