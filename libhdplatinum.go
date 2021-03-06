package libhdplatinum

import (
	"net"
	"fmt"
	"bufio"
	"time"
	"strings"
	"strconv"
	"log"
)

const UPDATE_COMMAND_FORMAT = "$pss%s-04-%03d-"
const END_COMMAND = "$rls"
const ROOM_PREFIX = "$cr"
const SHADE_PREFIX = "$cs"
const DELIMITER = "-"
const DATA_COMMAND = "$dat"
const DATA_DELIMITER = " "
const DATA_END = " $upd01-"

func connect(address string) (*bufio.Scanner, *bufio.Writer, *net.TCPConn) {
	tcpAddr, _ := net.ResolveTCPAddr("tcp4", address)
	conn, _ := net.DialTCP("tcp", nil, tcpAddr)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	scanner.Text() // get rid of the "Hunter Douglas" preamble from the read buffer

	return scanner, bufio.NewWriter(conn), conn
}

type Room struct {
	id string
	name string
	shades []Shade
}

func (r *Room) Id() string {
	return r.id
}

func (r *Room) Name() string {
	return r.name
}

func (r *Room) Shades() []Shade {
	return r.shades
}

type Shade struct {
	id string
	name string
	roomId string
	height int

	address string
}

func (s *Shade) Id() string {
	return s.id
}

func (s *Shade) Name() string {
	return s.name
}

func (s *Shade) Height() int {
	return s.height
}

func (s *Shade) RoomId() string {
	return s.roomId
}

func (s *Shade) SetHeight(h int) {
	_, writer, conn := connect(s.address)
	setHeightCommand := fmt.Sprintf(UPDATE_COMMAND_FORMAT, s.id, h)
	writer.WriteString(setHeightCommand)
	writer.Flush()
	writer.WriteString(END_COMMAND)
	writer.Flush()

	conn.Close()
	s.height = h
}

func GetShades(ip string, port int) ([]Shade) {
	address := ip + ":" + strconv.Itoa(port)
	data := getData(address)
	shades := []Shade{}
	for i, line := range data {
		if strings.HasPrefix(line, SHADE_PREFIX) {
			shadeTokens := strings.SplitN(line, DELIMITER, 4)
			id := strings.TrimPrefix(shadeTokens[0], SHADE_PREFIX)
			name := shadeTokens[3]
			roomId := shadeTokens[1]
			settingTokens := strings.SplitN(data[i+1], DELIMITER, 4)
			height, _ := strconv.Atoi(settingTokens[2])
			shade := Shade{id: id, name: name, roomId: roomId, height: height, address: address}
			shades = append(shades, shade)
		}
	}
	return shades
}

func GetRooms(ip string, port int) ([]Room) {
	address := ip + ":" + strconv.Itoa(port)
	data := getData(address)
	rooms := []Room{}
	allShades := GetShades(ip, port) // TODO: optimize. currently making 2 calls to getData
	for _, line := range data {
		if strings.HasPrefix(line, ROOM_PREFIX) {
			shadeTokens := strings.SplitN(line, DELIMITER, 4)
			id := strings.TrimPrefix(shadeTokens[0], ROOM_PREFIX)
			name := shadeTokens[3]
			roomShades := []Shade{}
			for _, shade := range allShades {
				if id == shade.RoomId() {
					roomShades = append(roomShades, shade)
				}
			}
			room := Room{id: id, name: name, shades: roomShades}
			rooms = append(rooms, room)
		}
	}
	return rooms
}

func getData(address string) ([]string) {
	scanner, writer, conn := connect(address)
	writer.WriteString(DATA_COMMAND)
	writer.Flush()

	scanSuccess := scanner.Scan()
	atEnd := false
	data := []string{}
	for scanSuccess && !atEnd {
		if (scanSuccess) {
			text := scanner.Text()
			tokens := strings.SplitN(text, DATA_DELIMITER, 2)
			if len(tokens) < 2 {
				log.Fatal("Error retrieving data from hd. Got ", text)
			}
			data = append(data, tokens[1])
			if strings.HasSuffix(text, DATA_END) {
				atEnd = true
			} else {
				scanSuccess = scanner.Scan()
			}
		}
	}

	conn.Close()
	return data
}
