#### AzDevOps Backup Job Flow Diagram


![Alt text](./image/azuredevops.png "Fluxo")


This project aims to download azuredevops projects from an ssh connection, download to a folder locally, zip and upload to an s3 bucket

#### For Executions

```
go run main.go

```

#### For Build Docker Image

```
docker build -f Dockerfile -t <nameusername>/<nameimage> .

```

#### For Run Docker Image

```
docker run -d <nameimage>

```
