package serializer

import "metoda/internal/app/ds"

type UserJSON struct {
	ID          uint   `json:"id"`
	Login       string `json:"login"`
	Password    string `json:"password,omitempty"`
	IsModerator bool   `json:"is_moderator"`
}

func UserToJSON(u ds.Users) UserJSON {
	return UserJSON{
		ID:          u.ID,
		Login:       u.Login,
		IsModerator: u.IsModerator,
	}
}

func UserFromJSON(j UserJSON) ds.Users {
	return ds.Users{
		Login:       j.Login,
		Password:    j.Password,
		IsModerator: j.IsModerator,
	}
}
