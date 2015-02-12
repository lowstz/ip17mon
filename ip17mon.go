package ip17mon

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io/ioutil"
	"net"
	"strings"
)

const Null = "N/A"

var (
	ErrInvalidIp = errors.New("invalid ip format")
	std          *Locator
)

func Init(dataFile string) (err error) {
	if std != nil {
		return
	}
	std, err = NewLocator(dataFile)
	return
}

func InitWithData(data []byte) {
	if std != nil {
		return
	}
	std = NewLocatorWithData(data)
	return
}

func Find(ipstr string) (*LocationInfo, error) {
	return std.Find(ipstr)
}

func FindByUint(ip uint32) *LocationInfo {
	return std.FindByUint(ip)
}

//-----------------------------------------------------------------------------

func NewLocator(dataFile string) (loc *Locator, err error) {
	data, err := ioutil.ReadFile(dataFile)
	if err != nil {
		return
	}
	loc = NewLocatorWithData(data)
	return
}

func NewLocatorWithData(data []byte) (loc *Locator) {
	loc = new(Locator)
	loc.init(data)
	return
}

type Locator struct {
	textData   []byte
	indexData1 []uint32
	indexData2 []int
	indexData3 []int
	index      []int
}

type LocationInfo struct {
	Country string
	Region  string
	City    string
	Isp     string
}

func (loc *Locator) Find(ipstr string) (info *LocationInfo, err error) {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		err = ErrInvalidIp
		return
	}
	info = loc.FindByUint(binary.BigEndian.Uint32([]byte(ip.To4())))
	return
}

func (loc *Locator) FindByUint(ip uint32) (info *LocationInfo) {
	idx := loc.findIndexOffset(ip, loc.index[ip>>24])
	off := loc.indexData2[idx]
	return newLocationInfo(loc.textData[off : off+loc.indexData3[idx]])
}

// binary search
func (loc *Locator) findIndexOffset(ip uint32, start int) int {
	end := len(loc.indexData1) - 1
	for start < end {
		mid := (start + end) / 2
		if ip > loc.indexData1[mid] {
			start = mid + 1
		} else {
			end = mid
		}
	}

	if loc.indexData1[end] >= ip {
		return end
	}

	return start
}

func (loc *Locator) init(data []byte) {
	textoff := int(binary.BigEndian.Uint32(data[:4]))

	loc.textData = data[textoff-1024:]

	loc.index = make([]int, 256)
	for i := 0; i < 256; i++ {
		off := 4 + i*4
		loc.index[i] = int(binary.LittleEndian.Uint32(data[off : off+4]))
	}

	nidx := (textoff - 4 - 1024 - 1024) / 8

	loc.indexData1 = make([]uint32, nidx)
	loc.indexData2 = make([]int, nidx)
	loc.indexData3 = make([]int, nidx)

	for i := 0; i < nidx; i++ {
		off := 4 + 1024 + i*8
		loc.indexData1[i] = binary.BigEndian.Uint32(data[off : off+4])
		loc.indexData2[i] = int(uint32(data[off+4]) | uint32(data[off+5])<<8 | uint32(data[off+6])<<16)
		loc.indexData3[i] = int(data[off+7])
	}
	return
}

func newLocationInfo(str []byte) *LocationInfo {
	fields := bytes.Split(str, []byte("\t"))
	if len(fields) != 4 {
		info := &LocationInfo{
			Country: "unknown",
			Region:  "unknown",
			City:    "unknown",
			Isp:     "unknown",
		}
		return info
	}
	info := &LocationInfo{
		Country: string(fields[0]),
		Region:  string(fields[1]),
		City:    string(fields[2]),
		Isp:     string(fields[3]),
	}

	if len(info.Country) == 0 {
		info.Country = Null
	}
	if len(info.Region) == 0 {
		info.Region = Null
	}
	if len(info.City) == 0 {
		info.City = Null
	}
	if idx := strings.Index(info.Isp, "/"); idx != -1 {
		info.Isp = info.Isp[:idx]
	}
	if len(info.Isp) == 0 {
		info.Isp = Null
	}
	return info
}
