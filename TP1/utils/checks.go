package utils

import "strconv"

func CheckCreateDeckError(error *string, err error, deck *int) {
	if *deck <= 0 {
		*error = "Le nombre de deck demandé est trop bas, 1 minimum"
	} else if *deck > 10 {
		*error = "Le nombre de deck demandé est trop haut, 10 maximum"
	}
	if err != nil {
		*error += err.Error()
	}
}

func CheckCard(card string) bool {
	if len(card) > 3 || len(card) < 1 {
		return false
	}

	rank := card[0]
	num, err := strconv.Atoi(string(rank))

	if err != nil || num < 0 || num > 13 {
		return false
	}

	if len(card) < 3 {
		suit := card[len(card)-1]
		if suit != 'd' && suit != 's' && suit != 'h' && suit != 'c' {
			return false
		}
	} else if len(card) == 3 {
		suit := card[1:]
		if suit != "sc" && suit != "dh" {
			return false
		}
	}
	return true
}
