package torrent_kad

import (
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
	Node *kademlia.Client
	peer Peer
}

func (this *Client) PutFile(filePath string) bool {
	infohash, exist, isDir := GenerateTorrentFile(filePath)
	if !exist {
		return false
	}
	this.peer.addPath(infohash, filePath, path.Base(filePath)+".torrent", isDir, pieceSize)
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
	return true
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
			info.infohash = part[12:42]
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
func (this *Client) GetFile(magnetLink string) bool {
	magnetlinkinfo := processMagnetLink(magnetLink)
	this.Node.Join(magnetlinkinfo.tracker) //todo:need to check whether it has joined before
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
	availablePieces := make([]intSet, availableServers.Len())
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
		if availablePieces[cnt] == nil {
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
	sort.Slice(&pieceOwnStat, func(i, j int) bool {
		return len(pieceOwnStat[i].servers) < len(pieceOwnStat[j].servers)
	})
	for _, i := range pieceOwnStat {
		client, e := rpc.Dial("tcp", i.servers[rand.Intn(len(i.servers))])
		if e != nil {

		}
	}

}
