## KPM Sparse-Checkout feature

## 1. Introduction
kpm which is the package management tool for KCL, does not support sparse-checkout, which means an issue arises when dealing with monorepos, which may contain many KCL packages. We need to get a solution to download specific packages instead of all in a monorepo.

## 2. Design research 
A solution to this problem lies onto using GithubAPIs for kpm. The API can be used to fetch repository contents and then processing the response data to list the sub-directories of a specific monorepo. 

## 3. User Interface 
The user will have to enter the command 
```
kpm add <github_repo_url>
```
to get the list of all the subdirectories(recursively all the directories which had a kcl.mod file in it). The user now has to toggle between the output subdirectories and press 'space' to select the ones which they want to keep. 

Considering the nginx-ingres module, on typing the command 

```
kpm add <url of nginx-ingres>
``` 
kpm will list the two subdirectories as 
- restrict-ingress-annotations
- restrict-ingress-paths

The user now has to select which package they want according to their project.

The experience for the user will be completely different than any other package manager installing packages or package subdirectories.

## 4. Implementation steps for the functionality
- setting up the Github API Access
- make a request to the endpoint `GET /repos/{owner}/{repo}/contents/{path}`
- parsing the JSON response to identify directories and subdirectories
- recursively fetch and process the contents of each directory to get a full list of subdirectories
- integrate it with the kpm code and update the kcl.mod accordingly

## 5. Integration and the use of go-getter to download the specific subdirectories

The repoUrl field in the struct `CloneOptions` in kpm/pkg/git/git.go will be given the subdir url accordingly, which then downloads each selected subdirectory one by one.

Also, the kcl.mod file will contain the list of all the subdirectories kept child to the main directory(if so).
