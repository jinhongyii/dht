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
	    Success bool
	    Header  Contact
    }
    
    type KVPair struct {
        Key string
        Val string
    }
    
    type StoreRequest struct {
	    Pair      KVPair
	    Header    Contact
	    Expire    time.Time
	    Replicate bool
    }
    
    type StoreReturn struct {
    	Success bool
    	Header  Contact
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
    
    
    RPCPing(Contact) -> Pingreturn
    RPCStore(StoreRequest) -> StoreReturn
    RPCFindNode(FindNodeRequest) -> (FindNodeReturn)
    RPCFindValue(FindValueRequest) -> (FindValueReturn)

The name of the type which supports RPC method: **Node**
