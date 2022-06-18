package GoNetReader

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
)

type NetReader struct {
	BufScaner     *bufio.Scanner // Scanner
	NetBufChannel chan string    // Channel
	_bufRes       []byte         // Buff
	_netError     error
	_q            chan byte // quit channel
}

func (reader *NetReader) FindPacksec(data []byte, atEOF bool) (advance int, token []byte, err error) { // scanner
	if atEOF || len(data) < 47 {
		reader._bufRes = data[:0]
		reader._netError = io.EOF
		return len(data), data, io.EOF
	}
	// find 1 pack
	for i := 0; i < len(data); i++ {
		if bytes.Equal(data[i:i+14], []byte("[StartPackage]")) { // Find StartPackage
			i += 14                                           // set offset by len([StartPackage])
			if bytes.Equal(data[i:i+8], []byte("[offset]")) { // find offset block
				for j := i + 9; j < len(data); j++ { // get offset number
					if bytes.Equal(data[j:j+11], []byte("[endoffset]")) { // find end offset block
						if s, err := strconv.Atoi(string(data[i+8 : j])); err == nil { // get offset number
							if bytes.Equal(data[j+11+s:j+23+s], []byte("[EndPackage]")) { // check end pack
								if len(data) > j+24+s { // if > 1 pack in buff
									reader._bufRes = data[j+23+s:] // add to buffer
									go reader.queueManager()       // start gorut
								} else {
									reader._bufRes = data[:0]
								}
								return j + 23 + s, data[j+11 : j+11+s], nil
							}
						}
					}
				}
			}
		}
	}
	reader._bufRes = data[:0]
	reader._netError = io.EOF
	return len(data), data, io.EOF
}

func (reader *NetReader) queueManager() { // manager buff
	if len(reader._bufRes) == 0 {
		return
	}
	secscan := bufio.NewScanner(bytes.NewReader(reader._bufRes))
	secscan.Split(reader.FindPacksec)
	secscan.Scan()
	if err := secscan.Err(); err != nil || reader._netError != nil {
		if err != nil && reader._netError == nil {
			reader._netError = err
		}
		reader._q <- 1
		return
	} else {
		reader.NetBufChannel <- secscan.Text()
	}
}

func (reader *NetReader) FindPack(data []byte, atEOF bool) (advance int, token []byte, err error) { // scanner
	if atEOF || len(data) < 47 {
		reader._netError = io.EOF
		return len(data), data, io.EOF
	}
	// find 1 pack
	for i := 0; i < len(data); i++ {
		if bytes.Equal(data[i:i+14], []byte("[StartPackage]")) { // Find StartPackage
			i += 14                                           // set offset by len([StartPackage])
			if bytes.Equal(data[i:i+8], []byte("[offset]")) { // find offset block
				for j := i + 9; j < len(data); j++ { // get offset number
					if bytes.Equal(data[j:j+11], []byte("[endoffset]")) { // find end offset block
						if s, err := strconv.Atoi(string(data[i+8 : j])); err == nil { // get offset number
							if bytes.Equal(data[j+11+s:j+23+s], []byte("[EndPackage]")) { // check end pack
								if len(data) > j+24+s { // if > 1 pack in buff
									reader._bufRes = append(reader._bufRes, data[j+23+s:]...) // add to buffer
									go reader.queueManager()                                  // start gorut
								}
								return j + 23 + s, data[j+11 : j+11+s], nil
							}
						}
					}
				}
			}
		}
	}
	reader._netError = io.EOF
	return len(data), data, io.EOF
}
func (reader *NetReader) scan() { // scanner
	reader.BufScaner.Scan()
	if err := reader.BufScaner.Err(); err != nil || reader._netError != nil {
		if err != nil && reader._netError == nil {
			reader._netError = err
		}
		reader._q <- 1
		return
	} else {
		reader.NetBufChannel <- reader.BufScaner.Text()
	}
}

func (reader *NetReader) ReadWithoutEmpty(Wchannel chan string) (string, error) {
	select {
	case Pack := <-Wchannel:
		{
			if Pack != "" && Pack != " " && len(Pack) > 0 { // check empty
				return Pack, nil
			}
			return reader.ReadWithoutEmpty(Wchannel)
		}
	case <-reader._q:
		{
			if reader._netError != nil {
				return "", reader._netError
			} else {
				return "", errors.New("Closed signal received")
			}
		}
	}
}

func (reader *NetReader) ReadPackage() (string, error) {
	reader.BufScaner.Split(reader.FindPack) // set scanner func
	go reader.scan()                        // start scanner
	return reader.ReadWithoutEmpty(reader.NetBufChannel)
}

func (reader *NetReader) NetRead(Connection net.Conn) (string, error) { // general read function
	if cap(reader.NetBufChannel) < 1 { // if chan not make
		reader.NetBufChannel = make(chan string, 5)
	}
	if cap(reader._q) < 1 {
		reader._q = make(chan byte, 2)
	}
	reader.BufScaner = bufio.NewScanner(Connection) // start new scanner
	return reader.ReadPackage()
}

func NewNetReader() *NetReader { // return reader
	nefr := NetReader{}
	return &nefr
}

func GetPackage(pack []byte) []byte { // func packing
	netoffset := []byte("[offset]" + strconv.Itoa((len(pack))) + "[endoffset]")
	return append([]byte("[StartPackage]"), append(append(netoffset, pack...), []byte("[EndPackage]")...)...)
}
