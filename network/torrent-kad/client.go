package torrent_kad

import (
	"dht"
	"fmt"
	"kademlia"
	"math/rand"
	"net/rpc"
	"os"
	"path"
	"sort"
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
	this.peer.addPath(infohash, filePath, filePath+".torrent", isDir, pieceSize)
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
	this.Node.Put(magnetlinkinfo.infohash, this.Node.Node_.RoutingTable.Ip)
	if !torrentGot {
		fmt.Println("torrent file not got")
		return false
	}
	torrentinfo := processTorrentFile(torrentFile)
	availablePieces := make([]IntSet, availableServers.Len())
	cnt := 0
	type stat struct {
		servers []string
		index   int
	}
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
	sort.Slice(pieceOwnStat, func(i, j int) bool {
		return len(pieceOwnStat[i].servers) < len(pieceOwnStat[j].servers)
	})
	ch := make(chan FilePiece, 1)
	for _, i := range pieceOwnStat {
		var randServer int
		for {
			randServer = rand.Intn(len(i.servers))
			if this.Node.Ping(i.servers[randServer]) {
				break
			} else {
				i.servers = append(i.servers[:randServer], i.servers[randServer+1:]...)
			}
		}
		go this.getPieceFromRemote(magnetlinkinfo.infohash, i.index, i.servers[randServer], torrentinfo["piece length"].(int), ch)
	}
	pieces := make([]FilePiece, 0)

	for i := 0; i < len(pieceOwnStat); i++ {
		pieceGot := <-ch
		pieces = append(pieces, pieceGot)
		if _, ok := this.peer.downloadingStatus[magnetlinkinfo.infohash]; ok {
			this.peer.downloadingStatus[magnetlinkinfo.infohash][pieceGot.index] = struct{}{}
		} else {
			this.peer.downloadingStatus[magnetlinkinfo.infohash] = make(IntSet)
			this.peer.downloadingStatus[magnetlinkinfo.infohash][pieceGot.index] = struct{}{}
		}
		if _, ok := this.peer.downloadedPiece[magnetlinkinfo.infohash]; ok {
			this.peer.downloadedPiece[magnetlinkinfo.infohash][pieceGot.index] = pieceGot.content
		} else {
			this.peer.downloadedPiece[magnetlinkinfo.infohash] = make(map[int][]byte)
			this.peer.downloadedPiece[magnetlinkinfo.infohash][pieceGot.index] = pieceGot.content
		}
	}
	sort.Slice(pieces, func(i, j int) bool {
		return pieces[i].index < pieces[j].index
	})
	file, _ := os.Create(path + "/" + torrentinfo["name"].(string))
	for i := 0; i < len(pieces)-1; i++ {
		file.Write(pieces[i].content)
	}
	file.Write(pieces[len(pieces)-1].content[:torrentinfo["length"].(int)%torrentinfo["piece length"].(int)])
	file.Close()
	this.peer.addPath(magnetlinkinfo.infohash, path+"/"+torrentinfo["name"].(string), magnetlinkinfo.fileName+".torrent", false, torrentinfo["piece length"].(int))
	delete(this.peer.downloadingStatus, magnetlinkinfo.infohash)
	delete(this.peer.downloadedPiece, magnetlinkinfo.infohash)
	return true
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
	err := client.Call("Peer.GetPiece", TorrentRequest{
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
	this.Node = main.NewNode(port)
	err := this.Node.Server.Register(&this.peer)
	if err != nil {
		fmt.Println(err)
	}
	this.Node.Run()
	this.peer.Init()
}
