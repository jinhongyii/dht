package torrent_kad

import (
	"errors"
	"io/ioutil"
	"net/rpc"
	"os"
)

type intSet map[int]struct{}

type Peer struct {
	infoHashMap       map[string]basicFileInfo
	downloadingStatus map[string]intSet
	server            *rpc.Server
}

func (this *Peer) Init() {
	this.server = rpc.NewServer()
}

type basicFileInfo struct {
	torrentPath string
	filePath    string
	isDir       bool
	pieceSize   int
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
func (this *Peer) GetPieceStatus(infohash string, availablePiece *intSet) error {
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
type Request struct {
	infohash string
	index    int
	length   int
}

func (this *Peer) GetPiece(request Request, content *[]byte) error {
	fileinfo := this.infoHashMap[request.infohash]
	*content = make([]byte, request.length)
	if !fileinfo.isDir {
		file, err := os.Open(fileinfo.filePath)
		if err != nil {
			return err
		}
		file.ReadAt(*content, int64(request.length*request.index))
	}
	return nil
}
func (this *Peer) addPath(infohash string, filePath string, torrentPath string, isDir bool, pieceSize int) {
	this.infoHashMap[infohash] = basicFileInfo{filePath: filePath, torrentPath: torrentPath, isDir: isDir, pieceSize: pieceSize}
}
