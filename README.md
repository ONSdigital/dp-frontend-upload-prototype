# dp-frontend-upload-prototype

### Prerequisites

- Go
- AWS Credentials
- Chrome or Firefox or Safari (This will not run on IE!!)

### How to run the server

- Clone the repository under $GOPATH/src/github.com/ONSdigital
- Run `make debug` to start the prototype server on port 3000

If you do not have AWS Credentials, the server will store the uploaded file locally
otherwise you will want to set the environment variables:

- AWS_ACCESS_KEY_ID=`ACCESS_KEY`
- AWS_SECRET_ACCESS_KEY=`SECRET_KEY`

before starting the server.

### How to use the prototype

- Navigate your browser to http://localhost:3000
- Click on the `Add Files` button to add file(s) you wish to upload
- Click on the `Start/Resume Upload` button to initiate the resumable multi part upload
- Uploads can be paused by clicking on the `Pause Upload` button, Resume by clicking the
`Start/Resume Upload` button.

The client (browser) will send the server the file in 5MB chunks. If the server is
running in `local` mode, the file will be pieced together and stored locally within
this repository. If the server is running in `AWS` mode, the server will initiate
a multi part upload to s3 and piece together the chunks as they are received,
until the last chunk is given to the server, when the multi part upload is marked as
complete and is made available in the s3 bucket named: `dp-frontend-upload-prototype`

### Testing the prototype

There are two ways to test the "resumability" of an upload:

1) If the server responds with an unexpected status, i.e 4.x.x or 5.x.x, then the
client will retry the upload after 5 seconds. To test this, kill the server half way
through a file upload. Keep the browser session open and restart the server after
a random amount of time. The client should automatically retry the upload.

2) User pausing/resuming of an upload. The obvious way to test this is to click on the
pause button half way through an upload. The upload progress will pause until the
Start/Resume button is clicked, when the client will resume uploading from the same
point.

This prototype currently has no concept of user sessions, so refreshing the page will
not automatically cause the upload to resume, however if the page is refreshed
part way through an upload, and the same file is selected again, the server will
recognise this file has been part uploaded and will skip over the chunks which have
already been uploaded and resume from the first chunk which is not in the cache.
When uploads are implemented in Florence, the user session should be able to recognise
that an upload was in progress when a user left the upload page and will be able to
resume the upload immediately when the user returns to that page.

If you wish to test the uploading of a large file, you can generate a random 1GB file
by running the command:

`head -c 1073741824 </dev/urandom >myfile`

If you wish to change the number of simultaneous requests to the server, change the value
of line 6 in javascript/upload.js to what you would like the number to be.

### Spike limitations and considerations

- The official AWS SDK was not used for this spike as the documentation is difficult
to read and does not seem idiomatic for the Go language. Instead,
http://godoc.org/gopkg.in/amz.v1/aws was used, as it was user friendly and well
documented.
- The s3 library can exhibit unexpected behaviour occasionally when handling
simultaneous uploads. When adding a part to a multi part upload the code will not
throw an error if the part is not added successfully, this means that when the client
has sent all its requests, then we have to check to see if all parts are actually in
the s3 multi cache. If this is not the case we have to tell the client to retry the
upload until all parts are in the cache. By default, the client will send 15
simultaneous requests to the server.
- It can take up to 10 seconds to receive a response from the server while it adds
parts to the s3 multi cache. This may be an unreasonable amount of time to wait, so it
could be worth either making the multi-upload to s3 asynchronous or to increase the
number of simultaneous requests to the server.
- Currently the unique id for the file is the file name. It may be worth creating a
unique id (uuid) to add in front of the file name to avoid conflicts when uploading
files with the same name.
- We should probably assert the checksum (MD5) is the same before and after the upload
to avoid any data corruption during the data upload.
- Increasing the number of simultaneous requests does improve the time taken for a file
to upload. For example, a 1GB file with 3 simultaneous requests will take ~ 20 minutes to
upload - increasing this number to 15 simultaneous requests results in the file taking
around 10 minutes to upload. It is not yet known what the limit is before the server
begins to slow after a given number of simultaneous requests.
