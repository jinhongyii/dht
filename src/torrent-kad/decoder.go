package torrent_kad

import (
	"bufio"
	"errors"
	"strconv"
	"strings"
)

type Bdecoder struct {
	reader *bufio.Reader
}

func (this *Bdecoder) PeekByte() (byte, error) {
	ch, err := this.reader.Peek(1)
	if err != nil {
		return 0, err
	} else {
		return ch[0], nil
	}
}

func (this *Bdecoder) GetString() (string, error) {
	strlen, err := this.reader.ReadBytes(':')
	if err != nil {
		return "", err
	}
	length, e := strconv.Atoi(string(strlen[:len(strlen)-1]))
	if e != nil {
		return "", e
	}
	str := make([]byte, length)
	_, e = this.reader.Read(str)
	//fmt.Println("read ",l)
	if e != nil {
		return "", e
	}
	return string(str), nil
}

func (this *Bdecoder) GetInt() (int, error) {
	_, err := this.reader.ReadByte()
	if err != nil {
		return 0, err
	}
	numstr, e := this.reader.ReadBytes('e')
	if e != nil {
		return 0, e
	}
	num, err := strconv.Atoi(string(numstr[:len(numstr)-1]))
	return num, err
}
func (this *Bdecoder) GetList() ([]interface{}, error) {
	_, err := this.reader.ReadByte()
	if err != nil {
		return nil, err
	}
	ret := make([]interface{}, 0)
	for {
		item, e := this.Get()
		if e != nil {
			break
		} else {
			ret = append(ret, item)
		}
	}
	return ret, nil
}
func (this *Bdecoder) GetDict() (map[string]interface{}, error) {
	_, err := this.reader.ReadByte()
	if err != nil {
		return nil, err
	}
	ret := make(map[string]interface{})
	for {
		key, err1 := this.Get()
		if err1 != nil {
			break
		}
		val, err2 := this.Get()
		if err2 != nil {
			break
		} else {
			ret[key.(string)] = val
		}
	}
	return ret, nil
}
func (this *Bdecoder) Get() (interface{}, error) {
	peek, err := this.PeekByte()
	if err != nil {
		return nil, err
	}
	switch peek {
	case 'i':
		return this.GetInt()
	case 'l':
		return this.GetList()
	case 'd':
		return this.GetDict()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return this.GetString()
	case 'e':
		this.reader.ReadByte()
		return nil, errors.New("end")
	default:
		return nil, errors.New("wrong format")
	}
}
func NewDecoder(str string) *Bdecoder {
	decoder := new(Bdecoder)
	strreader := strings.NewReader(str)
	decoder.reader = bufio.NewReaderSize(strreader, len(str))
	return decoder
}
