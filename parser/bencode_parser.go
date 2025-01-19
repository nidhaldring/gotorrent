package parser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type BencodeDict map[string]any

func ParseBencode(bencode string) (BencodeDict, error) {
	if len(bencode) == 0 {
		return nil, errors.New("Expected a bencode string got an empty string instead")
	}

	if bencode[0] != 'd' && bencode[len(bencode)-1] != 'e' {
		return nil, errors.New(fmt.Sprintf("bencode should start with 'd' and ends with 'e' got beg=%c & end=%c", bencode[0], bencode[len(bencode)-1]))
	}

	i := 1
	return consumeDict(bencode, &i)
}

/*
* PLEASE NOTE ALL "consumeXXX" FUNC WILL POSITION "pos" AFTER THE PARSED VALUE
 */

func consumeDict(bencode string, pos *int) (BencodeDict, error) {
	dict := make(BencodeDict)

	// check if it's an empty dict
	if *pos+1 < len(bencode)-1 && bencode[*pos+1] == 'e' {
		*pos++
		return dict, nil
	}

	for *pos < len(bencode)-1 && bencode[*pos] != 'e' {
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

	return dict, nil
}

func consumeString(bencode string, pos *int) (string, error) {
	sep := strings.IndexByte(bencode[*pos:], ':')
	if sep == -1 {
		return "", errors.New(fmt.Sprintf("Expected a separator around index %d", *pos))
	}

	strLen, err := strconv.Atoi(bencode[*pos:sep])
	if err != nil || strLen <= 0 {
		return "", errors.New(fmt.Sprintf("Expected str len to be a positive number got %s instead", bencode[*pos:sep]))
	}

	str := bencode[sep+1 : sep+1+strLen]
	*pos += (sep - *pos) + strLen + 1
	return str, nil
}

func consumeInt(bencode string, pos *int) (int, error) {
	intEnding := strings.IndexByte(bencode[*pos:], 'e')
	if intEnding == -1 {
		return 0, errors.New(fmt.Sprintf("Invalid integer found at %d", *pos))
	}

	num, err := strconv.Atoi(bencode[*pos+1 : intEnding])
	if err != nil {
		return 0, errors.New(fmt.Sprintf("Invalid integer found at %d", *pos))
	}

	*pos = intEnding + 1
	return num, nil

}

func consumeList(bencode string, pos *int) ([]any, error) {
	arr := make([]any, 0)

	*pos++ // skip the 'l'
	for bencode[*pos] != 'e' {
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

	return arr, nil
}

/* END OF consumeXXX functions */
