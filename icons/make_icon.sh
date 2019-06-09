#/bin/sh

if [ -z "$GOPATH" ]; then
    echo GOPATH environment variable not set
    exit
fi

#if [ ! -e "$GOPATH/bin/2goarray" ]; then
#    echo "Installing 2goarray..."
#    go get github.com/cratonica/2goarray
#    if [ $? -ne 0 ]; then
#        echo Failure executing go get github.com/cratonica/2goarray
#        exit
#    fi
#fi

go build gen/2goarray.go

OUTPUT=icons_data.go
echo Generating $OUTPUT
echo "//+build linux" > $OUTPUT
echo >> $OUTPUT
echo "package icons" >> $OUTPUT
for file in $(ls icon_files/*.png)
do
  echo Processing $file
  cat "$file" | ./2goarray $(basename "$file" .png) >> $OUTPUT
  if [ $? -ne 0 ]; then
    echo Failure processing $File
    exit 1
  fi
done

echo Finished
