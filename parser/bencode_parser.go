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
	if *pos >= len(bencode) {
		return nil, errors.New(fmt.Sprintf("Position='%d' is greater than bencode length='%d'", *pos, len(bencode)))
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
