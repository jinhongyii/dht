package torrent_kad

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os"
	"strconv"
	"sync"
)

type IntSet map[int]struct{}

type Peer struct {
	infoHashMap       map[string]basicFileInfo //todo:save a fstream in the map
	downloadingStatus map[string]IntSet
	server            *rpc.Server
}

func (this *Peer) Init() {
	this.server = rpc.NewServer()
	this.downloadingStatus = make(map[string]IntSet)
	this.infoHashMap = make(map[string]basicFileInfo)
}

type basicFileInfo struct {
	torrentPath string
	filePath    string
	isDir       bool
	pieceSize   int
	fileio      *os.File
	fileMux     sync.Mutex
}

func (this *Peer) GetTorrentFile(infoHash string, torrent *[]byte) error {
	torrentPath := this.infoHashMap[infoHash].torrentPath
	content, err := ioutil.ReadFile(torrentPath)
	if err != nil {
		return err
	} else {
		*torrent = content
	}
	return nil
}
func (this *Peer) GetPieceStatus(infohash string, availablePiece *IntSet) error {
	_, ok := this.infoHashMap[infohash]
	if !ok {
		return errors.New("no such file")
	}
	pieces, exist := this.downloadingStatus[infohash]
	if !exist {
		*availablePiece = pieces
	} else {
		*availablePiece = nil
	}
	return nil
}

//todo:change network
type TorrentRequest struct {
	Infohash string
	Index    int
	Length   int
}

func (this *Peer) GetFilePiece(request TorrentRequest, content *[]byte) error {
	if _, ok := this.downloadingStatus[request.Infohash]; ok {
		fileinfo := this.infoHashMap[request.Infohash]
		path := fileinfo.filePath
		fileinfo.fileMux.Lock()
		*content, _ = ioutil.ReadFile(path + "/" + request.Infohash + "/" + strconv.Itoa(request.Index) + ".piece")
		fileinfo.fileMux.Unlock()
	} else {
		fileinfo := this.infoHashMap[request.Infohash]
		*content = make([]byte, request.Length)
		if !fileinfo.isDir {
			fileinfo.fileMux.Lock()
			file := fileinfo.fileio
			file.ReadAt(*content, int64(request.Length*request.Index))
			fileinfo.fileMux.Unlock()
		}
	}
	return nil
}
func (this *Peer) GetDirectoryPiece(request TorrentRequest, content *[]byte) error {
	fileinfo := this.infoHashMap[request.Infohash]
	if _, ok := this.downloadingStatus[request.Infohash]; ok {

		path := fileinfo.filePath
		fileinfo.fileMux.Lock()
		*content, _ = ioutil.ReadFile(path + "/" + request.Infohash + "/" + strconv.Itoa(request.Index) + ".piece")
		fileinfo.fileMux.Unlock()
	} else {
		fileinfo.fileMux.Lock()
		torrentbyte, err := ioutil.ReadFile(fileinfo.torrentPath)
		if err != nil {
			return nil
		}
		torrentinfo := processTorrentFile(torrentbyte)
		files := torrentinfo["files"].([]interface{})
		cnt := 0
		i := 0
		for i = 0; i < len(files); i++ {

			fileLength := files[i].(map[string]interface{})["length"].(int)
			if cnt+fileLength > request.Length*request.Index {
				break
			}
			cnt += fileLength
		}
		start := request.Length*request.Index - cnt
		var end int
		for i < len(files) {
			flag := false
			if cnt+files[i].(map[string]interface{})["length"].(int) >= request.Length*(request.Index+1) {
				end = request.Length*(request.Index+1) - cnt
				flag = true
			} else {
				end = files[i].(map[string]interface{})["length"].(int)
			}
			fmt.Println(fileinfo.filePath + "/" + assembleString(files[i].(map[string]interface{})["path"].([]interface{})))
			fileContent, _ := ioutil.ReadFile(fileinfo.filePath + "/" + assembleString(files[i].(map[string]interface{})["path"].([]interface{})))
			*content = append(*content, fileContent[start:end]...)
			cnt += files[i].(map[string]interface{})["length"].(int)
			i++
			start = 0
			if flag {
				break
			}
		}
		fileinfo.fileMux.Unlock()
	}
	//if len(*content)<fileinfo.pieceSize {
	//	tmp:=make([]byte,fileinfo.pieceSize-len(*content))
	//	*content=append(*content,tmp...)
	//}
	return nil
}
func assembleString(dirs []interface{}) string {
	res := ""
	for _, str := range dirs {
		res += str.(string) + "/"
	}
	if res != "" {
		res = res[:len(res)-1]
	}
	return res
}
func (this *Peer) addFileInfo(infohash string, filePath string, torrentPath string, isDir bool, pieceSize int) {
	var file *os.File
	this.infoHashMap[infohash] = basicFileInfo{filePath: filePath, torrentPath: torrentPath, isDir: isDir, pieceSize: pieceSize,
		fileio: file, fileMux: sync.Mutex{}}

}
func (this *Peer) openFileio(infohash string) {
	tmp := this.infoHashMap[infohash]
	tmp.fileio, _ = os.Open(this.infoHashMap[infohash].filePath)
	this.infoHashMap[infohash] = tmp
}
