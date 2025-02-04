package decoder

import (
	"errors"
	"fmt"
	"gotorrent/utils"
	"os"
	"strconv"
	"strings"
)

type TorrentFile struct {
	Announce     string
	AnnounceList [][]string
	CreatedBy    string
	CreationDate int
	Encoding     string
	Info         TorrentInfo
}

type TorrentInfo struct {
	Length      int
	Name        string
	PieceLength int
	Pieces      string
}

func (t TorrentFile) String() string {
	return fmt.Sprintf(`
  [announce]: %s
  [announce list]: %+v
  [created by]: %s
  [creation date]: %s
  [encoding]: %s
  [info]: {
    [length]: %d
    [name]: %s
    [pieces length]: %d
    [pieces]: [...]
  }
  `, t.Announce, t.AnnounceList, t.CreatedBy, t.CreationDate, t.Encoding, t.Info.Length, t.Info.Name, t.Info.PieceLength)
}

// This is type alias that does not declare a new type
// thus it's possible for me later to easily type cast this to an actual dict
// read more here: https://stackoverflow.com/questions/61247864/what-is-the-difference-between-type-alias-and-type-definition-in-go
type BencodeDict = map[string]any

func DecodeTorrentFile(filename string) (*TorrentFile, error) {
	b, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	dict, err := Decode(string(b))
	if err != nil {
		return nil, err
	}

	var t TorrentFile
	err = utils.MapToStruct(dict, &t)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func Decode(bencode string) (BencodeDict, error) {
	if len(bencode) == 0 {
		return nil, errors.New("Expected a bencode string got an empty string instead")
	}

	i := 0
	dict, err := consumeDict(bencode, &i)
	if err != nil {
		return nil, err
	}

	return dict, nil
}

/*
* PLEASE NOTE ALL "consumeXXX" FUNC WILL POSITION "pos" AFTER THE PARSED VALUE
 */

func consumeDict(bencode string, pos *int) (BencodeDict, error) {
	if *pos >= len(bencode) {
		return nil, errors.New(fmt.Sprintf("Position='%d' is greater than bencode length='%d'", *pos, len(bencode)))
	}

	if bencode[*pos] != 'd' {
		return nil, errors.New(fmt.Sprintf("Expected a dict start at pos %d got %s instead", *pos, string(bencode[*pos])))
	}

	dict := make(BencodeDict)

	*pos++ // skip the d
	for *pos < len(bencode) && bencode[*pos] != 'e' {
		key, err := consumeString(bencode, pos)
		if err != nil {
			return nil, err
		}

		var val any
		switch bencode[*pos] {
		case 'd':
			d, err := consumeDict(bencode, pos)
			if err != nil {
				return nil, err
			}
			val = d

		case 'l':
			l, err := consumeList(bencode, pos)
			if err != nil {
				return nil, err
			}
			val = l

		case 'i':
			num, err := consumeInt(bencode, pos)
			if err != nil {
				return nil, err
			}
			val = num

		default:
			str, err := consumeString(bencode, pos)
			if err != nil {
				return nil, err
			}

			val = str
		}

		dict[key] = val
	}

	if *pos < len(bencode) && bencode[*pos] == 'e' {
		*pos += 1
		return dict, nil
	}

	return nil, errors.New(fmt.Sprintf("Expected dict to end with 'e' at position '%d'", *pos))
}

func consumeList(bencode string, pos *int) ([]any, error) {
	if *pos >= len(bencode) {
		return nil, errors.New(fmt.Sprintf("Position='%d' is greater than bencode length='%d'", *pos, len(bencode)))
	}

	if bencode[*pos] != 'l' {
		return nil, errors.New(fmt.Sprintf("Expected a list start at pos %d got %s instead", *pos, string(bencode[*pos])))
	}

	arr := make([]any, 0)
	*pos++ // skip the 'l'
	for *pos < len(bencode) && bencode[*pos] != 'e' {
		var val any
		switch bencode[*pos] {
		case 'i':
			n, err := consumeInt(bencode, pos)
			if err != nil {
				return nil, err
			}
			val = n

		case 'l':
			l, err := consumeList(bencode, pos)
			if err != nil {
				return nil, err
			}
			val = l

		case 'd':
			d, err := consumeDict(bencode, pos)
			if err != nil {
				return nil, err
			}
			val = d

		default:
			s, err := consumeString(bencode, pos)
			if err != nil {
				return nil, err
			}

			val = s
		}

		arr = append(arr, val)
	}

	if *pos < len(bencode) && bencode[*pos] == 'e' {
		*pos += 1
		return arr, nil
	}

	return nil, errors.New(fmt.Sprintf("Expected list to end with 'e' at position '%d'", *pos))
}

func consumeString(bencode string, pos *int) (string, error) {
	if *pos >= len(bencode) {
		return "", errors.New(fmt.Sprintf("Position='%d' is greater than bencode length='%d'", *pos, len(bencode)))
	}

	sep := strings.IndexByte(bencode[*pos:], ':')
	if sep == -1 {
		return "", errors.New(fmt.Sprintf("Expected a separator around index %d", *pos))
	}
	sep += *pos

	strLen, err := strconv.Atoi(bencode[*pos:sep])
	if err != nil || strLen <= 0 {
		return "", errors.New(fmt.Sprintf("Expected str len to be a positive number got %s instead at pos %d", bencode[*pos:sep], *pos))
	}

	str := bencode[sep+1 : sep+1+strLen]
	*pos += (sep - *pos) + strLen + 1
	return str, nil
}

func consumeInt(bencode string, pos *int) (int, error) {
	if *pos >= len(bencode) {
		return 0, errors.New(fmt.Sprintf("Position='%d' is greater than bencode length='%d'", *pos, len(bencode)))
	}

	if bencode[*pos] != 'i' {
		return 0, errors.New(fmt.Sprintf("Expected start of int got %s at index %d instead", string(bencode[*pos]), *pos))
	}

	intEnding := strings.IndexByte(bencode[*pos:], 'e')
	if intEnding == -1 {
		return 0, errors.New(fmt.Sprintf("Int with no delimiter found at %d", *pos))
	}
	intEnding += *pos

	num, err := strconv.Atoi(bencode[*pos+1 : intEnding])
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Cannot convert int at %d got '%s' from (%d, %d) instead", *pos, bencode[*pos+1:intEnding], *pos+1, intEnding))
	}

	*pos = intEnding + 1
	return num, nil
}

/* END OF consumeXXX functions */
