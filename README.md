# dedupe
A memory efficient, concurrent duplicate file finder written in go.  There are probably a thousand
duplicate file finder tools on github. This is mine.

## build
```shell
> git checkout https://github.com/victortrac/dedupe.git
> cd dedupe
> go build -o dedupe main.go
```
## usage
```shell
> ./dedupe <dir1> <dir2> ... > duplicates.log

> cat duplicates.log
/volume2/media/Pictures-3/2010/06/12/IMG_1507.JPG, Pictures/2010/06/12/IMG_1507.JPG
/volume2/media/Pictures-3/2010/06/12/IMG_1509.JPG, Pictures/2010/06/12/IMG_1509.JPG
/volume2/media/Pictures-3/2010/06/12/IMG_1516.JPG, Pictures/2010/06/12/IMG_1516.JPG
...
```
You need to provide at least one directory for it to scan.

I then use a little bash script to delete the files:
```shell
> cat delete.sh
#!/bin/bash
while read line; do
  file_to_delete=$(echo "$line" | cut -d ',' -f1)
  echo "Deleting $file_to_delete"
  rm "$file_to_delete"
done < duplicates.log
```

## how
1. It walks the supplied directories and builds a hashmap of each file
2. Before inserting into the hashmap, it checks to see if the hash already exists
3. If there's already a matching md5, it does a byte-by-byte comparison to make sure it's not a simple hash collision
4. If the bytes match, it determines that the files are identical

## why
I had multiple directories on my NAS with tens of thousands of photos taken over the years 
in various stages of being 'backed up'. I needed a way to quickly identify the duplicate files 
and remove them.  It also had to run natively on my synology NAS without using a lot of memory.

