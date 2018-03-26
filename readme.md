# LKFCoder

LKFCoder is a utility for encoding and decoding files of LKF format.

## Building
The utility has no third-party dependencies and compiled on any platform, where the golang compiler is available. For get and install LKFCoder, run the following command:

	go get github.com/kvark128/LKFCoder

## Using
The first argument of program specifies the required action: decode or encode or version.

* decode - specifies that the LKFCoder should decode lkf files to mp3 format.
* encode - specifies that the LKFCoder should encode mp3 files to lkf format.
* version - shows the version LKFCoder.

The second argument specifies path to the file or directory that requires processing.
If the second argument is not specified, the current work directory is used.
When specifying a directory, all files in all its subdirectories will be processed.
The processed files are determined by extension. lkf are decoded to mp3 or mp3 are encoded to lkf.

For example, if the book of the lkf format is located on the path C:\MyBook, then to convert it to the mp3 format, run the following command:

	LKFCoder decode C:\MyBook

The result of the work is written to the source file, after which he changes extension.
