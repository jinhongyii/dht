package main

import "os"

type AppServer struct {
	FileNameMap map[string]string
}

func (this *AppServer) GetFile(fileHash string, file *[]byte) error {
	fileName := this.FileNameMap[fileHash]
	bytes, err := getBytesFromFile(fileName)
	if err != nil {
		return err
	} else {
		*file = bytes
		return nil
	}
}

type Getfilerequest struct {
	FileHash   string
	Index      int
	Totpartnum int
}

func (this *AppServer) GetFilePart(request Getfilerequest, file *[]byte) error {
	fileName := this.FileNameMap[request.FileHash]
	bytes, err := getBytesFromFilePart(fileName, request.Index, request.Totpartnum)
	if err != nil {
		return err
	} else {
		*file = bytes
		return nil
	}
}
func (this *AppServer) GetFileExistence(filehash string, exist *bool) error {
	fileAddr, ok := this.FileNameMap[filehash]
	f, err := os.Open(fileAddr)
	if err == nil {
		f.Close()
	}
	*exist = ok && err == nil
	return nil
}
func NewServer() *AppServer {
	server := new(AppServer)
	server.FileNameMap = make(map[string]string)
	return server
}
