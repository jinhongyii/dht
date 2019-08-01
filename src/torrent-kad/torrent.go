package torrent_kad

import (
	"crypto/sha1"
	"io"
	"io/ioutil"
	"kademlia"
	"log"
	"math/big"
	"net/rpc"
	"os"
	"path"
	"strconv"
	"strings"
)

const pieceSize = 1 << 18

type Torrent struct {
	dht_node kademlia.Client
	server   rpc.Server
}

//if eof return true
func getFilePartHash(f *os.File, size int, pieceid int) (string, bool) {
	buf := make([]byte, size)
	_, err := f.ReadAt(buf, int64(pieceid)*int64(size))
	hash := sha1.Sum(buf)
	return string(hash[:]), err != nil
}
func getDirectoryHash(builder *strings.Builder, dirPath string, buf *[]byte) {
	files, _ := ioutil.ReadDir(dirPath)
	for _, file := range files {
		if file.IsDir() {
			getDirectoryHash(builder, dirPath+"/"+file.Name(), buf)
		} else {
			f, _ := os.Open(dirPath + "/" + file.Name())
			for {
				localbuf := make([]byte, pieceSize-len(*buf))
				read_len, err := f.Read(localbuf)
				if err != nil || read_len < len(localbuf) {
					*buf = append(*buf, localbuf[:read_len]...)
					break
				}
				localbuf = append(*buf, localbuf...)
				*buf = (*buf)[:0]
				hash := sha1.Sum(localbuf)
				builder.Write(hash[:])
			}
		}
	}
}
func recursiveGetFileinfo(stringBuilder *strings.Builder, prefix []string, dirPath string) {
	files, _ := ioutil.ReadDir(dirPath)
	for _, file := range files {
		prefix = append(prefix, file.Name())
		if file.IsDir() {
			recursiveGetFileinfo(stringBuilder, prefix, dirPath+"/"+file.Name())
			prefix = prefix[:len(prefix)-1]
		} else {
			stringBuilder.WriteString("d6:length")
			stringBuilder.WriteString("i" + strconv.Itoa(int(file.Size())) + "e")
			stringBuilder.WriteString("4:path")
			stringBuilder.WriteString("l")
			for _, p := range prefix {
				stringBuilder.WriteString(strconv.Itoa(len(p)) + ":" + p)
			}
			stringBuilder.WriteString("ee")
			prefix = prefix[:len(prefix)-1]
		}
	}
}

/**
@returns info-hash
*/
func GenerateTorrentFile(filepath string) (string, bool, bool) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", false, false
	}
	file.Close()
	var stringbuilder strings.Builder
	stringbuilder.WriteString("d4:name")
	originalPath := filepath
	filename := path.Base(filepath)
	f, _ := os.Stat(filepath)
	isDirectory := f.IsDir()
	stringbuilder.WriteString(strconv.Itoa(len(filename)) + ":" + filename)
	stringbuilder.WriteString("12:piece length")
	stringbuilder.WriteString("i" + strconv.Itoa(pieceSize) + "e")
	stringbuilder.WriteString("6:pieces")

	if isDirectory {
		hashbuilder := strings.Builder{}
		GetDirectoryHash(&hashbuilder, filepath)
		hash := hashbuilder.String()
		stringbuilder.WriteString(strconv.Itoa(len(hash)) + ":" + hash)
		stringbuilder.WriteString("5:filesl")
		prefix := make([]string, 0)
		recursiveGetFileinfo(&stringbuilder, prefix, filepath)
		stringbuilder.WriteString("e")
	} else {
		GetPiecedFileHash(&stringbuilder, filepath)
		stringbuilder.WriteString("6:length")
		stringbuilder.WriteString("i" + strconv.Itoa(int(f.Size())) + "e")
	}
	stringbuilder.WriteString("e")
	newTorrentFile, _ := os.Create(originalPath + ".torrent")
	newTorrentFile.WriteString(stringbuilder.String())
	newTorrentFile.Close()

	return GetCompleteFileHash(originalPath + ".torrent"), true, isDirectory
}

func GetDirectoryHash(hashbuilder *strings.Builder, filepath string) {
	buf := make([]byte, 0)
	getDirectoryHash(hashbuilder, filepath, &buf)
	if len(buf) != 0 {
		tmp := sha1.Sum(buf)
		hashbuilder.Write(tmp[:])
	}
}

func GetPiecedFileHash(stringbuilder *strings.Builder, filepath string) {
	file, _ := os.Open(filepath)
	defer file.Close()
	hashtot := ""
	for i := 0; ; i++ {
		hash, over := getFilePartHash(file, pieceSize, i)
		hashtot += hash
		if over {
			break
		}
	}
	stringbuilder.WriteString(strconv.Itoa(len(hashtot)) + ":" + hashtot)
}
func GetCompleteFileHash(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	return new(big.Int).SetBytes(h.Sum(nil)).Text(16)
}
