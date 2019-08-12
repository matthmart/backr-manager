# Backr Manager

Backr Manager is a tool created to manage backup files stored into an object storage solution. 

## Purpose

This tool manages the lifecycle of backup files, using some fine-grained control rules. You may want to keep some recent files of the last 3 days and some older files (e.g. 15 days at least), to be able to turn back time.

One of the key features of the tool is to detect some issues with the backup routine. Backr Manager is able to monitor and send alerts when expected files are missing or when a file size is a lot smaller than the previous file.

## Architecture

The main entities are *files* and *projects*. Projects are configured by the user, allowing to define lifecycle rules for files stored into a same folder. So a folder is linked to a project. Files are stored in an object storage service like S3. Each similar file is expected to be stored in a specific folder. File uploading is not the responsability of Backr Manager.

So the requirements/assumptions are:

- similar files must be stored in a folder
- a project is linked to a folder and lifecycle rules will be applied to the files of this folder

When the daemon is starting, the process manager runs periodically and checks for file changes in each configured project. It detects potential issues (small size, missing file, etc), send alerts if needed and remove files not needed anymore (if the rules are fulfilled).

A gRPC API is exposed to communicate with the process manager. It allows to manage projects, list files, get a temporary URL to download a file. The user must be authenticated to interact with the API. User management is also integrated to the API.

A CLI client is available to interact with the API.

