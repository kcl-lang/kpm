## KPM Sparse-Checkout feature

## 1. Introduction
kpm which is the package management tool for KCL, does not support sparse-checkout, which means an issue arises when dealing with monorepos, which may contain many KCL packages. We need to get a solution to download specific packages instead of all in a monorepo.

## 2. Design research 
A solution to this problem lies onto using GithubAPIs for kpm. The API can be used to fetch repository contents and then processing the response data to list the sub-directories of a specific monorepo. On listing the subdirectories, we can make the user toggle between different packages and press *space* to select the packages they want according to their project.

## 3. Implementation steps 
- setting up the Github API Access
- make a request to the endpoint `GET /repos/{owner}/{repo}/contents/{path}`
- parsing the JSON response to identify directories and subdirectories
- recursively fetch and process the contents of each directory to get a full list of subdirectories
- integrate it with the kpm code

we will use Go's standard libraries for HTTP requests and JSON decoding, might need to handle additional error and edge cases.