package models

type Note struct {
	ID      int
	Content string `json:"content" binding:"required"`
}

// func GetAllBook(b *[]Book) (err error) {
// 	if err = config.DB.Find(b).Error; err != nil {
// 		return err
// 	}
// 	return nil
// }

// func AddNewNote(n *Note) (err error) {
// 	if err = Config.db.Create(n).Error; err != nil {
// 		return err
// 	}
// 	return nil
// }

// func GetOneNote(n *Note, id int) (err error) {
// 	if err := config.DB.Where("id = ?", id).First(n).Error; err != nil {
// 		return err
// 	}
// 	return nil
// }

// func PutOneNote(n *Note, id string) (err error) {
// 	fmt.Println(n)
// 	config.DB.Save(n)
// 	return nil
// }

// func DeleteBook(b *Book, id string) (err error) {
// 	config.DB.Where("id = ?", id).Delete(b)
// 	return nil
// }

func (b *Note) TableName() string {
	return "notes"
}
