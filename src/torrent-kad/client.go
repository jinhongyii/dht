package torrent_kad

import (
	"common"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"kademlia"
	"math/rand"
	"net/rpc"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

const initialNodeIp = "localhost:2000"

type Client struct {
	Node   *kademlia.Client
	peer   Peer
	Joined bool
}

func (this *Client) PutFile(filePath string) (string, bool) {
	if !this.Joined {
		this.Node.Join(initialNodeIp)
		this.Joined = true
	}
	infohash, exist, isDir := GenerateTorrentFile(filePath)
	if !exist {
		return "", false
	}
	this.peer.addFileInfo(infohash, filePath, filePath+".torrent", isDir, pieceSize)
	this.peer.openFileio(infohash)
	this.Node.Put(infohash, this.Node.Node_.RoutingTable.Ip)
	magnetLinkBuilder := strings.Builder{}
	magnetLinkBuilder.WriteString("magnet:?xt=urn:btih:")
	magnetLinkBuilder.WriteString(infohash)
	magnetLinkBuilder.WriteString("&dn=")
	fileName := path.Base(filePath)
	fileName = strings.ReplaceAll(fileName, "%", "%25")
	fileName = strings.ReplaceAll(fileName, "&", "%26")
	fileName = strings.ReplaceAll(fileName, " ", "%20")
	magnetLinkBuilder.WriteString(fileName)
	magnetLinkBuilder.WriteString("&tr=")
	magnetLinkBuilder.WriteString(initialNodeIp)
	fmt.Println("magnetLink:", magnetLinkBuilder.String())
	return magnetLinkBuilder.String(), true
}

type magnetLinkInfo struct {
	infohash string
	fileName string
	tracker  string
}

func processMagnetLink(magnetLink string) magnetLinkInfo {
	info := magnetLinkInfo{}
	magnetBody := magnetLink[8:]
	parts := strings.Split(magnetBody, "&")
	for _, part := range parts {
		part = strings.ReplaceAll(part, "%20", " ")
		part = strings.ReplaceAll(part, "%26", "&")
		part = strings.ReplaceAll(part, "%25", "%")
		if part[:3] == "xt=" {
			if part[7:11] != "btih" {
				fmt.Println("wrong encoding method")
			}
			info.infohash = part[12:52]
		} else if part[:3] == "dn=" {
			info.fileName = part[3:]
		} else if part[:3] == "tr=" {
			info.tracker = part[3:]
		}
	}
	return info
}

//type TorrentInfo struct {
//	suggestedName string
//	pieceLength   int
//	pieces []string
//	isdir bool
//	length int
//	files []BasicFileInfo
//}
func processTorrentFile(torrentFile []byte) map[string]interface{} {
	decoder := NewDecoder(string(torrentFile))
	torrentinfo, err := decoder.Get()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return torrentinfo.(map[string]interface{})
}
func (this *Client) GetFile(magnetLink string, path string) bool {
	magnetlinkinfo := processMagnetLink(magnetLink)
	if !this.Joined {
		if !this.Node.Join(magnetlinkinfo.tracker) {
			fmt.Println("join fail")
			return false
		}
		this.Joined = true //todo:need to check whether it has joined before
	}
	availableServers, ok := this.Node.Get(magnetlinkinfo.infohash)
	if !ok {
		return false
	}
	torrentGot := false
	var torrentFile []byte
	for server := range availableServers {
		client, err := rpc.Dial("tcp", server)
		if err != nil {
			fmt.Println(err)
			continue
		}
		err = client.Call("Peer.GetTorrentFile", magnetlinkinfo.infohash, &torrentFile)
		client.Close()
		if err != nil {
			fmt.Println(err)
			continue
		}
		newTorrentFile, _ := os.Create(magnetlinkinfo.fileName + ".torrent")
		newTorrentFile.Write(torrentFile)
		newTorrentFile.Close()
		torrentGot = true
		break
	}
	if !torrentGot {
		fmt.Println("torrent file not got")
		return false
	}
	torrentinfo := processTorrentFile(torrentFile)
	_, isdir := torrentinfo["files"]
	if !isdir {
		this.peer.downloadingStatus[magnetlinkinfo.infohash] = make(IntSet)
		this.peer.addFileInfo(magnetlinkinfo.infohash, path+"/"+torrentinfo["name"].(string),
			magnetlinkinfo.fileName+".torrent", isdir, torrentinfo["piece length"].(int))
		this.Node.Put(magnetlinkinfo.infohash, this.Node.Node_.RoutingTable.Ip)
		this.getPieces(availableServers, torrentinfo, &magnetlinkinfo, path)

		file, _ := os.Create(path + "/" + torrentinfo["name"].(string))
		recvFile := assembleFile(path, len(torrentinfo["pieces"].(string))/20, magnetlinkinfo.infohash,
			torrentinfo["length"].(int), torrentinfo["piece length"].(int))
		file.Write(recvFile)
		file.Close()
		this.peer.openFileio(magnetlinkinfo.infohash)
		delete(this.peer.downloadingStatus, magnetlinkinfo.infohash)
		os.RemoveAll(path + "/" + magnetlinkinfo.infohash)
	}
	return true
}

type stat struct {
	servers []string
	index   int
}

func assembleFile(path string, piecenum int, infohash string, totlen int, piece_length int) []byte {
	content := make([]byte, 0)

	for i := 0; i < piecenum-1; i++ {
		buf, _ := ioutil.ReadFile(path + "/" + infohash + "/" + strconv.Itoa(i) + ".piece")
		content = append(content, buf...)
	}
	buf, _ := ioutil.ReadFile(path + "/" + infohash + "/" + strconv.Itoa(piecenum-1) + ".piece")
	content = append(content, buf[:totlen%piece_length]...)
	return content
}
func (this *Client) getPieces(availableServers kademlia.Set, torrentinfo map[string]interface{}, magnetlinkinfo *magnetLinkInfo, filepath string) {
	availablePieces := make([]IntSet, availableServers.Len())
	cnt := 0
	pieceOwnStat := make([]stat, len(torrentinfo["pieces"].(string))/20)
	for i := range pieceOwnStat {
		pieceOwnStat[i].index = i
		pieceOwnStat[i].servers = make([]string, 0)
	}
	for server := range availableServers {
		client, e := rpc.Dial("tcp", server)
		if e != nil {
			cnt++
			continue
		}
		client.Call("Peer.GetPieceStatus", magnetlinkinfo.infohash, &availablePieces[cnt])
		if len(availablePieces[cnt]) == 0 {
			for i := range pieceOwnStat {
				pieceOwnStat[i].servers = append(pieceOwnStat[i].servers, server)
			}
		} else {
			for piece := range availablePieces[cnt] {
				pieceOwnStat[piece].servers = append(pieceOwnStat[piece].servers, server)
			}
		}
		client.Close()
		cnt++
	}
	var original = make([]stat, len(pieceOwnStat))
	copy(original, pieceOwnStat)
	sort.Slice(pieceOwnStat, func(i, j int) bool {
		return len(pieceOwnStat[i].servers) < len(pieceOwnStat[j].servers)
	})
	ch := make(chan FilePiece, 1)
	for _, i := range pieceOwnStat {
		this.sendGetPieceRequest(&i, i.index, magnetlinkinfo, torrentinfo, ch)
	}
	os.MkdirAll(filepath+"/"+magnetlinkinfo.infohash, 0666)

	for i := 0; i < len(pieceOwnStat); i++ {
		pieceGot := <-ch
		hash := sha1.Sum(pieceGot.content)
		if string(hash[:]) != torrentinfo["pieces"].(string)[pieceGot.index*20:(pieceGot.index+1)*20] {
			this.sendGetPieceRequest(&original[pieceGot.index], pieceGot.index, magnetlinkinfo, torrentinfo, ch)
			i--
		}
		tmpFile, _ := os.Create(filepath + "/" + magnetlinkinfo.infohash + "/" + strconv.Itoa(pieceGot.index) + ".piece")
		tmpFile.Write(pieceGot.content)
		tmpFile.Close()

		this.peer.downloadingStatus[magnetlinkinfo.infohash][pieceGot.index] = struct{}{}

	}
	return
}

func (this *Client) sendGetPieceRequest(serverStat *stat, index int, magnetlinkinfo *magnetLinkInfo, torrentinfo map[string]interface{}, ch chan FilePiece) {
	var randServer int
	for {
		randServer = rand.Intn(len(serverStat.servers))
		if this.Node.Ping(serverStat.servers[randServer]) {
			break
		} else {
			serverStat.servers = append(serverStat.servers[:randServer], serverStat.servers[randServer+1:]...)
		}
	}
	go this.getPieceFromRemote(magnetlinkinfo.infohash, index, serverStat.servers[randServer], torrentinfo["piece length"].(int), ch)
}

type FilePiece struct {
	index   int
	content []byte
}

func (this *Client) getPieceFromRemote(infohash string, pieceno int, ip string, length int, recv chan FilePiece) {
	client, e := rpc.Dial("tcp", ip)
	if e != nil {
		fmt.Println("get piece ", pieceno, " from ", ip, " failed")
		return
	}
	var content = make([]byte, 0)
	err := client.Call("Peer.GetFilePiece", TorrentRequest{
		Infohash: infohash,
		Index:    pieceno,
		Length:   length,
	}, &content)
	defer client.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	recv <- FilePiece{pieceno, content}
}

func (this *Client) Init(port int) {
	this.Node = common.NewNode(port)
	err := this.Node.Server.Register(&this.peer)
	if err != nil {
		fmt.Println(err)
	}
	this.Node.Run()
	this.peer.Init()
}
