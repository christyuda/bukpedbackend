package menu

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gocroot/helper/atdb"
	"github.com/gocroot/helper/tiket"
	"github.com/whatsauth/itmodel"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Mengecek apakah pesan adalah nomor menu, jika nomor menu maka akan mengubahnya menjadi keyword
func MenuSessionHandler(msg *itmodel.IteungMessage, db *mongo.Database) string {
	// check apakah nomor adalah admin atau user untuk menentukan startmenu
	var startmenu string
	if !tiket.IsAdmin(msg.Phone_number, db) {
		startmenu = "menu"
	} else {
		startmenu = "adminmenu"
	}
	fmt.Println("Startmenu determined:", startmenu) // Debugging startmenu

	// check apakah ada session, klo ga ada insert session baru
	Sesdoc, ses, err := CheckSession(msg.Phone_number, db)
	if err != nil {
		fmt.Println("Error checking session:", err) // Debug error check session
		return err.Error()
	}
	fmt.Println("Session exists:", ses, "Session document:", Sesdoc) // Debug session state and document

	if !ses { // jika tidak ada session atau session=false maka return menu utama user dan update session isi list nomor menunya
		reply, err := GetMenuFromKeywordAndSetSession(startmenu, Sesdoc, db)
		if err != nil {
			fmt.Println("Error getting menu from keyword:", err) // Debug error in menu retrieval
			return err.Error()
		}
		fmt.Println("Reply on new session:", reply) // Debug reply from the new session setup
		return reply
	}

	// jika ada session maka cek menu
	fmt.Println("Existing session found, checking menu") // Debug existing session found

	// check apakah pesan integer
	menuno, err := strconv.Atoi(msg.Message)
	if err == nil { // kalo pesan adalah nomor
		fmt.Println("Message is a valid number:", menuno) // Debug if message is a valid number
		for _, menu := range Sesdoc.Menulist {            // looping di menu list dari session
			fmt.Println("Checking menu number:", menu.No, "with keyword:", menu.Keyword) // Debug each menu item in session

			if menuno == menu.No { // jika nomor menu sama dengan nomor yang ada di pesan
				fmt.Println("Menu number matches:", menuno) // Debug when menu number matches
				reply, err := GetMenuFromKeywordAndSetSession(menu.Keyword, Sesdoc, db)
				if err != nil {
					fmt.Println("Error getting menu for keyword:", menu.Keyword) // Debug error for keyword menu retrieval
					msg.Message = menu.Keyword
					return ""
				}
				fmt.Println("Reply for matching menu:", reply) // Debug reply after matching menu
				return reply
			}
		}
		fmt.Println("Menu number not found:", menuno) // Debug if menu number not found
		return "Mohon maaf nomor menu yang anda masukkan tidak ada di daftar menu"
	}
	fmt.Println("Message is not a valid number:", msg.Message) // Debug if message is not a number
	// kalo pesan bukan nomor return kosong
	return ""
}

// check session udah ada atau belum kalo sudah ada maka refresh session
func CheckSession(phonenumber string, db *mongo.Database) (session Session, result bool, err error) {
	session, err = atdb.GetOneDoc[Session](db, "session", bson.M{"phonenumber": phonenumber})
	session.CreatedAt = time.Now()
	session.PhoneNumber = phonenumber
	if err != nil { //insert session klo belum ada
		_, err = db.Collection("session").InsertOne(context.TODO(), session)
		if err != nil {
			return
		}
	} else { //jika sesssion udah ada
		//refresh waktu session dengan waktu sekarang
		_, err = atdb.DeleteManyDocs(db, "session", bson.M{"phonenumber": phonenumber})
		if err != nil {
			return
		}
		_, err = db.Collection("session").InsertOne(context.TODO(), session)
		if err != nil {
			return
		}
		result = true
	}
	return
}

func GetMenuFromKeywordAndSetSession(keyword string, session Session, db *mongo.Database) (msg string, err error) {
	dt, err := atdb.GetOneDoc[Menu](db, "menu", bson.M{"keyword": keyword})
	if err != nil {
		return
	}
	atdb.UpdateOneDoc(db, "session", bson.M{"phonenumber": session.PhoneNumber}, bson.M{"list": dt.List})
	msg = dt.Header + "\n"
	for _, item := range dt.List {
		msg += strconv.Itoa(item.No) + ". " + item.Konten + "\n"
	}
	msg += dt.Footer
	return
}

func InjectSessionMenu(menulist []MenuList, phonenumber string, db *mongo.Database) error {
	_, err := atdb.UpdateOneDoc(db, "session", bson.M{"phonenumber": phonenumber}, bson.M{"list": menulist})
	return err
}
