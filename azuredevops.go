package main

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/core"
	"github.com/microsoft/azure-devops-go-api/azuredevops/git"
)

const (
    AWS_S3_REGION = ""
    AWS_S3_BUCKET = ""
	organizationUrl = ""
	personalAccessToken = ""
)


func main() {
	
connection := azuredevops.NewPatConnection(organizationUrl, personalAccessToken)
ctx := context.Background()

coreClient, gitClient := Client(ctx, connection)
getProject, getRepo := AzDevOpsConnection(coreClient, ctx, gitClient)
session, err := S3Connection()

index := 0
totalRepositorios := 0

for getProject != nil {
	
	for _, teamProjectReference := range (*getProject).Value {
		log.Printf("############ NOME PROJETO ############## [%v] = %v", index, *teamProjectReference.Name)
		index++
		totalRepositorios = GetRepositories(teamProjectReference, getRepo, err, gitClient, ctx, totalRepositorios)
	}
	
	if getProject.ContinuationToken != "" {

		projectArgs := core.GetProjectsArgs{
			ContinuationToken: &getProject.ContinuationToken,
		}
		getProject, err = coreClient.GetProjects(ctx, projectArgs)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		getProject = nil
	}
}

	log.Printf("Iniciando Processo de Zip")
	if err := ZipRepositories("./Repositorios", "repositories.zip"); err != nil {
        log.Fatal(err)
    }

	log.Printf("Iniciando Processo de Upload para o S3")
	err = UploadFileToS3(session, "repositories.zip")
	if err != nil {
		log.Fatal(err)
	}	

	log.Printf("Total de Repositorios no AzureDevOps é = %v", totalRepositorios)
	log.Printf("Total de Projetos no AzureDevOps é = %v", index)
}

func GetRepositories(teamProjectReference core.TeamProjectReference, getRepo *[]git.GitRepository, err error, gitClient git.Client, ctx context.Context, totalRepositorios int) int {
	repoValue := git.GetRepositoriesArgs{
		Project:        teamProjectReference.Name,
		IncludeAllUrls: newTrue(),
	}

	getRepo, err = gitClient.GetRepositories(ctx, repoValue)

	for _, nameRepo := range *getRepo {
		log.Printf("Repositorio = %v", *nameRepo.Name)
		log.Printf("Repositorio = %v", *nameRepo.SshUrl)
		log.Printf("Repositorio = %v", *nameRepo.WebUrl)
		totalRepositorios++

		 CloneRepositoy(nameRepo)

	}
	return totalRepositorios
}

func S3Connection() (*session.Session, error) {
	session, err := session.NewSession(&aws.Config{Region: aws.String(AWS_S3_REGION)})

	if err != nil {
		log.Fatal(err)
	}
	return session, err
}

func AzDevOpsConnection(coreClient core.Client, ctx context.Context, gitClient git.Client) (*core.GetProjectsResponseValue, *[]git.GitRepository) {
	getProject, err := coreClient.GetProjects(ctx, core.GetProjectsArgs{})
	getRepo, err := gitClient.GetRepositories(ctx, git.GetRepositoriesArgs{})

	if err != nil {
		log.Fatal(err)
	}
	return getProject, getRepo
}

func Client(ctx context.Context, connection *azuredevops.Connection) (core.Client, git.Client) {
	coreClient, err := core.NewClient(ctx, connection)
	gitClient, err := git.NewClient(ctx, connection)

	if err != nil {
		log.Fatal(err)
	}
	return coreClient, gitClient
}

func CloneRepositoy(nameRepo git.GitRepository) {
	cmd := exec.Command("git", "clone", *nameRepo.SshUrl)
	cmd.Dir = "./Repositorios"

	err := cmd.Run()
	if err != nil {
		log.Printf("Houve um erro ao clonar o repositorio")
	}
}

func UploadFileToS3(session *session.Session, uploadFileDir string) error {
    
    upFile, err := os.Open(uploadFileDir)
    if err != nil {
        return err
    }
    defer upFile.Close()
    
    upFileInfo, _ := upFile.Stat()
    var fileSize int64 = upFileInfo.Size()
    fileBuffer := make([]byte, fileSize)
    upFile.Read(fileBuffer)
    
    _, err = s3.New(session).PutObject(&s3.PutObjectInput{
        Bucket:               aws.String(AWS_S3_BUCKET),
        Key:                  aws.String(uploadFileDir),
        ACL:                  aws.String("private"),
        Body:                 bytes.NewReader(fileBuffer),
        ContentLength:        aws.Int64(fileSize),
        ContentType:          aws.String(http.DetectContentType(fileBuffer)),
        ContentDisposition:   aws.String("attachment"),
        ServerSideEncryption: aws.String("AES256"),
    })
    return err
}

func newTrue() *bool {
    b := true
    return &b
}

func ZipRepositories(source, target string) error {

    f, err := os.Create(target)
    if err != nil {
        return err
    }
    defer f.Close()

    writer := zip.NewWriter(f)
    defer writer.Close()

    return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        header, err := zip.FileInfoHeader(info)
        if err != nil {
            return err
        }

        header.Method = zip.Deflate

        header.Name, err = filepath.Rel(filepath.Dir(source), path)
        if err != nil {
            return err
        }
        if info.IsDir() {
            header.Name += "/"
        }

        headerWriter, err := writer.CreateHeader(header)
        if err != nil {
            return err
        }

        if info.IsDir() {
            return nil
        }

        f, err := os.Open(path)
        if err != nil {
            return err
        }
        defer f.Close()

        _, err = io.Copy(headerWriter, f)
        return err
    })
}








