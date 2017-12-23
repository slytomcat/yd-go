#/bin/sh

if [ -z "$GOPATH" ]; then
    echo GOPATH environment variable not set
    exit
fi

if [ ! -e "$GOPATH/bin/2goarray" ]; then
    echo "Installing 2goarray..."
    go get github.com/cratonica/2goarray
    if [ $? -ne 0 ]; then
        echo Failure executing go get github.com/cratonica/2goarray
        exit
    fi
fi

OUTPUT=icons.go
echo Generating $OUTPUT
echo "//+build linux" > $OUTPUT
echo >> $OUTPUT
for file in $(ls *.png)
do
  echo Processing $file
  cat "$file" | $GOPATH/bin/2goarray $(basename "$file" .png) icon >> $OUTPUT
  if [ $? -ne 0 ]; then
    echo Failure processing $File
    exit 1
  fi
done
echo Finished
