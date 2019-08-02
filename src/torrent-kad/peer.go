package torrent_kad

import (
	"errors"
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

//func GetDirectoryPiece(request TorrentRequest,content*[]byte)error{
//
//}
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
