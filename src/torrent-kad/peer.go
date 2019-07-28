package torrent_kad

import (
	"errors"
	"io/ioutil"
	"net/rpc"
	"os"
)

type IntSet map[int]struct{}

type Peer struct {
	infoHashMap       map[string]basicFileInfo //todo:save a fstream in the map
	downloadingStatus map[string]IntSet
	downloadedFile    map[string]*[]FilePiece
	server            *rpc.Server
}

func (this *Peer) Init() {
	this.server = rpc.NewServer()
	this.downloadingStatus = make(map[string]IntSet)
	this.downloadedFile = make(map[string]*[]FilePiece)
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
	infohash string
	index    int
	length   int
}

func (this *Peer) GetPiece(request TorrentRequest, content *[]byte) error {
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
