package utils

import "strconv"

// CheckCreateDeckError Vérifie que le deck n'a pas d'erreur
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

// CheckCard Vérifie que la carte n'a pas d'erreur
func CheckCard(card string) bool {
	if len(card) > 3 || len(card) < 1 {
		return false
	}

	rank := int(card[0])
	num, err := strconv.Atoi(string(rank))

	if err != nil || num < 0 || num > 13 {
		return false
	}

	if len(card) < 3 {

		if num > 13 {
			return false
		}

		suit := card[len(card)-1]
		if suit != 'd' && suit != 's' && suit != 'h' && suit != 'c' {
			return false
		}
	} else if len(card) == 3 {
		if _, err = strconv.Atoi(card[0:2]); err != nil {
			suit := card[1:]
			if suit != "jr" && suit != "jn" {
				return false
			}
		} else {
			num, _ = strconv.Atoi(card[0:2])
		}
	}
	return true
}
