# Protocol Assigned

----------

## Torrent Structure

    Dict {
        (string) name
        (int) piece length
        (string) piece
        (int) length(single file)
        (list)files(folder) {
            Dict {
                (int) length
                (list) path{string}
            }
        }
    }

## Magnet Link Structure

    magnet:?xt=urn:btih:<info-hash>&dn=<name>&tr=<tracker-url>

这里的tr是指带领你加入dht的服务器

## Kademlia RPC Protocols

    type Contact struct {
    	Id *big.Int //server's Id
    	Ip string   //server's Ip
    }
    
    type PingReturn struct {
    	Header  Contact
	Success bool
    }
    
    type KVPair struct {
        Key string
        Val string
    }
    
    type StoreRequest struct {
    	    Header    Contact
	    Pair      KVPair
	    Expire    time.Time
    }
    
    type StoreReturn struct {
        Header  Contact
    	Success bool
    }
    
    type FindNodeRequest struct {
    	Header Contact
    	Id     *big.Int
    }
    
    type FindNodeReturn struct {
    	Header  Contact
    	Closest Contacts
    }
    
    type FindValueRequest struct {
    	Header Contact
    	HashId *big.Int
    	Key    string
    }
    
    type Set map[string]struct{}
    
    type FindValueReturn struct {
    	Header  Contact
    	Closest []Contact
    	Val     Set
    }
    
    
    RPCPing(Contact) -> PingReturn
    RPCStore(StoreRequest) -> StoreReturn
    RPCFindNode(FindNodeRequest) -> (FindNodeReturn)
    RPCFindValue(FindValueRequest) -> (FindValueReturn)

The name of the type which supports RPC method: **Node**
## torrent client protocol
the name of rpc server is Peer

    type TorrentRequest struct {
        Infohash string
        Index    int
        Length   int
    }
    
    type IntSet map[int]struct{}
    
    GetTorrentFile(*string)-> []byte
    GetPieceStatus(*string)-> IntSet
    GetPiece(*TorrentRequest)->[]byte

    
