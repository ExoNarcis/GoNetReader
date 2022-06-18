package GoNetReader

import (
	"bufio"
	"bytes"
	"net"
	"strconv"
)

type NetReader struct {
	BufScaner     *bufio.Scanner // Scanner
	NetBufChannel chan string    // Channel
	_bufRes       []byte         // Buff
	_netError     error
}

func (reader *NetReader) splitShoter() ([]string, error) { // splitter
	for i := 0; i < len(reader._bufRes); i++ {
		if bytes.Equal(reader._bufRes[i:i+14], []byte("[StartPackage]")) { // Find StartPackage
			i += 14                                                     // set offset by len([StartPackage])
			if bytes.Equal(reader._bufRes[i:i+8], []byte("[offset]")) { // find offset block
				for j := i + 9; j < len(reader._bufRes); j++ { // get offset number
					if bytes.Equal(reader._bufRes[j:j+11], []byte("[endoffset]")) { // find end offset block
						if s, err := strconv.Atoi(string(reader._bufRes[i+8 : j])); err == nil { // get offset number
							if bytes.Equal(reader._bufRes[j+11+s:j+23+s], []byte("[EndPackage]")) { // check end pack
								rString := string(reader._bufRes[j+11 : j+11+s]) // getstr pack
								if len(reader._bufRes) > j+24+s {                // if > 1 pack in buff
									reader._bufRes = reader._bufRes[j+23+s:] // free 1 pack
									shot, err := reader.splitShoter()        // rec call
									if err != nil {
										return []string{rString}, nil // error in rec
									} else {
										return append([]string{rString}, shot...), nil //push arrays and return
									}
								} else {
									return []string{rString}, nil // return pack
								}
							}
						}
					}
				}
			}
		}
	}
	reader._bufRes = reader._bufRes[:0] // clear buff
	return nil, bufio.ErrFinalToken
}

func (reader *NetReader) queueManager() { // manager buff
	if len(reader._bufRes) == 0 {
		return
	}
	strigSlise, err := reader.splitShoter() // get packages in buff
	if err != nil {
		reader._bufRes = reader._bufRes[:0] // clear buff
		reader._netError = err
		return
	} else {
		for i := 0; i < len(strigSlise); i++ {
			reader.NetBufChannel <- strigSlise[i] // send to channel pack
		}
		reader._bufRes = reader._bufRes[:0] // clear buff
	}
}

func (reader *NetReader) FindPack(data []byte, atEOF bool) (advance int, token []byte, err error) { // scanner
	if atEOF || len(data) == 0 {
		return len(data), data, bufio.ErrFinalToken
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
	return len(data), data, bufio.ErrFinalToken
}
func (reader *NetReader) scan() { // scanner
	reader.BufScaner.Scan()
	reader.NetBufChannel <- reader.BufScaner.Text()
}

func (reader *NetReader) ReadWithoutEmpty(Wchannel chan string) (string, error) {
	if err := reader.BufScaner.Err(); err != nil {
		return "", err
	}
	if reader._netError != nil {
		return "", reader._netError
	}
	Pack := <-Wchannel
	if Pack != "" && Pack != " " && len(Pack) > 0 { // check empty
		return Pack, nil
	}
	return reader.ReadWithoutEmpty(Wchannel)
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
	reader.BufScaner = bufio.NewScanner(Connection) // start new scanner
	return reader.ReadPackage()
}

func NewNetReader() *NetReader { // return reader
	nefr := NetReader{}
	//go nefr.queueManager()
	return &nefr
}

func GetPackage(pack []byte) []byte { // func packing
	netoffset := []byte("[offset]" + strconv.Itoa((len(pack))) + "[endoffset]")
	return append([]byte("[StartPackage]"), append(append(netoffset, pack...), []byte("[EndPackage]")...)...)
}
