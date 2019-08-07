package torrent_kad

import (
	"bufio"
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
	basicLogger       *os.File
}

func (this *Peer) Init() {
	this.server = rpc.NewServer()
	this.downloadingStatus = make(map[string]IntSet)
	this.infoHashMap = make(map[string]basicFileInfo)
	this.basicLogger, _ = os.OpenFile("torrent.log", os.O_RDWR|os.O_CREATE, 0666)
	bufReader := bufio.NewReader(this.basicLogger)
	for {
		infohash, err1 := bufReader.ReadString('\n')

		if err1 != nil {
			break
		}
		infohash = infohash[:len(infohash)-1]
		torrentpath, _ := bufReader.ReadString('\n')
		torrentpath = torrentpath[:len(torrentpath)-1]
		filepath, _ := bufReader.ReadString('\n')
		filepath = filepath[:len(filepath)-1]
		fileio, _ := os.Open(filepath)
		isdir, _ := bufReader.ReadString('\n')
		isdir = isdir[:len(isdir)-1]
		piecesize, err3 := bufReader.ReadString('\n')
		piecesize = piecesize[:len(piecesize)-1]
		piecesizeint, _ := strconv.Atoi(piecesize)
		if err3 != nil {
			break
		}
		this.infoHashMap[infohash] = basicFileInfo{torrentPath: torrentpath, filePath: filepath,
			isDir: isdir == "1", pieceSize: piecesizeint, fileio: fileio}

	}
}

type basicFileInfo struct {
	torrentPath string
	filePath    string
	isDir       bool
	pieceSize   int
	fileio      *os.File
	fileMux     sync.Mutex
}

func (this *Peer) GetTorrentFile(infoHash *string, torrent *[]byte) error {
	torrentPath := this.infoHashMap[*infoHash].torrentPath
	content, err := ioutil.ReadFile(torrentPath)
	if err != nil {
		return err
	} else {
		*torrent = content
	}
	return nil
}
func (this *Peer) GetPieceStatus(infohash *string, availablePiece *IntSet) error {
	_, ok := this.infoHashMap[*infohash]
	if !ok {
		return errors.New("no such file")
	}
	pieces, exist := this.downloadingStatus[*infohash]
	if exist {
		*availablePiece = pieces
	} else {
		tmp := make(IntSet)
		tmp[-1] = struct{}{}
		*availablePiece = tmp
	}
	return nil
}

//todo:change network
type TorrentRequest struct {
	Infohash string
	Index    int
	Length   int
}

func (this *Peer) GetPiece(request *TorrentRequest, content *[]byte) error {
	fileinfo := this.infoHashMap[request.Infohash]
	if _, ok := this.downloadingStatus[request.Infohash]; ok {

		path := fileinfo.filePath
		fileinfo.fileMux.Lock()
		*content, _ = ioutil.ReadFile(path + "/" + request.Infohash + "/" + strconv.Itoa(request.Index) + ".piece")
		fileinfo.fileMux.Unlock()
	} else {
		if !fileinfo.isDir {
			*content = make([]byte, request.Length)
			if !fileinfo.isDir {
				fileinfo.fileMux.Lock()
				file := fileinfo.fileio
				file.ReadAt(*content, int64(request.Length*request.Index))
				fileinfo.fileMux.Unlock()
			}
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
				fileContent := make([]byte, pieceSize)
				f, _ := os.Open(fileinfo.filePath + "/" + assembleString(files[i].(map[string]interface{})["path"].([]interface{})))
				f.ReadAt(fileContent, int64(start))
				f.Close()
				*content = append(*content, fileContent[:end-start]...)
				cnt += files[i].(map[string]interface{})["length"].(int)
				i++
				start = 0
				if flag {
					break
				}
			}
			fileinfo.fileMux.Unlock()
		}
	}
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
	var isdirStr string
	if isDir {
		isdirStr = "1"
	} else {
		isdirStr = "0"
	}

	this.basicLogger.WriteString(infohash + "\n" + torrentPath + "\n" + filePath + "\n" +
		isdirStr + "\n" + strconv.Itoa(pieceSize) + "\n")
}
func (this *Peer) openFileio(infohash string) {
	tmp := this.infoHashMap[infohash]
	tmp.fileio, _ = os.Open(this.infoHashMap[infohash].filePath)
	this.infoHashMap[infohash] = tmp
}
