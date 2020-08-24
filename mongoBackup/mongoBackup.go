package mongoBackup

import (
	"context"
	"fmt"
	"github.com/codeskyblue/go-sh"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"strconv"
	"time"
)

type MongoProvider struct {
	client     *mongo.Client
	Connection *Connection
	Backup     map[string]*Backup
}

type Connection struct {
	Host       string `json:"connection_host"`
	Port       string `json:"connection_port, omitempty"`
	DbName     string `json:"connection_dbname"`
	Username   string `json:"connection_username, omitempty"`
	Password   string `json:"connection_password, omitempty"`
	BackupName string `json:"connection_backupname, omitempty"`
}

type Backup struct {
	Name string `json:"backup_name,omitempty" bson:"backup_name"`
	Time string `json:"backup_time,omitempty" bson:"backup_time"`
	Path string `json:"backup_path,omitempty" bson:"backup_path"`
	Size int64  `json:"backup_size,omitempty" bson:"backup_size"`
}

var uri string

func NewMongoClient(host string, port string, dbName string, user string, pass string, backupName string) *MongoProvider {
	// mongodb://[username:password@]host1[:port1][,...hostN[:portN]][/[defaultauthdb][?options]]
	uri = fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=admin", user, pass, host, dbName)
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	log.Printf("Connected to %s on %s!\n", dbName, host)
	err = client.Connect(ctx)
	return &MongoProvider{
		client: client,
		Connection: &Connection{
			Host:       host,
			Port:       port,
			DbName:     dbName,
			Username:   user,
			Password:   pass,
			BackupName: backupName,
		},
	}
}

func (m *MongoProvider) DbBackup() error {
	now := time.Now()
	backupTime := strconv.FormatInt(now.UTC().Unix(), 10)
	backupName := fmt.Sprintf("%s-%s", m.Connection.DbName, backupTime)
	path := fmt.Sprintf("/tmp/%s.gz", backupName)
	err := m.mongoDump(uri, path)
	if err != nil {
		log.Fatal(err)
	}
	var dbSize int64
	time.Sleep(time.Second * 1) // TODO: Add a file detection?
	file, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	dbSize = file.Size() / 1048576
	backup := Backup{
		Name: backupName,
		Time: now.Format(time.RFC3339),
		Path: fmt.Sprintf("/tmp/%s.aes", backupName),
		Size: dbSize,
	}
	backupEncrypt(backup.Name, m.Connection.Password)
	log.Println("Encryption Completed")
	m.mongoBackupInsert(backup.Name, backup.Time, m.Connection.DbName, backup.Path, dbSize)
	fmt.Println(backup)
	return nil
}

func (m *MongoProvider) DbRestore() error {
	backupDecrypt(m.Connection.BackupName, m.Connection.Password)
	log.Println("Decryption Completed")
	m.mongoRestore()
	return nil
}

func (m *MongoProvider) DbBackupQuery() error {
	//var backups []*Backup // TODO: Load backup into object
	coll := m.client.Database("backup").Collection("backups")
	//filter := bson.D{{Key:"name", Value:"test"}} // TODO: Only show db-specific backups
	//filter := bson.E{Key:"name", Value: bson.D{
	//	{"$regex", primitive.Regex{Pattern:"^test", Options:"i"}},
	//}}
	//cursor, err := coll.Find(context.Background(), bson.E{Key:"name", Value: primitive.Regex{Pattern:"^[a-z].*", Options:"i"}})
	cursor, err := coll.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for cursor.Next(context.TODO()) {
		count++
		elem := &bson.D{}
		if err := cursor.Decode(&elem); err != nil {
			log.Fatal(err)
		}
		//backups = Backup{
		//	Name: ,
		//	Time: "",
		//	Path: "",
		//	Size: 0,
		//}
		//backups = append(backups, elem)
		log.Println(elem)
	}
	log.Printf("You have %d backups\n", count)
	return nil
}

func (m *MongoProvider) mongoDump(uri string, path string) error {
	log.Printf("Backing up via %s\n", uri)
	dump := fmt.Sprintf("mongodump --forceTableScan --uri=%s --archive=%s --gzip", uri, path)
	out, err := sh.Command("/bin/sh", "-c", dump).SetTimeout(time.Second * 300).CombinedOutput()
	if err != nil {
		log.Fatalf("mongodump failed due to %s", out)
	}
	log.Printf("%s dumped successfully!\n", path)
	return nil
}

func (m *MongoProvider) mongoRestore() {
	log.Printf("Restoring %s via %s\n", m.Connection.BackupName, uri)
	restore := fmt.Sprintf("mongorestore --drop --uri=%s --archive=/tmp/%s.gz --gzip", uri, m.Connection.BackupName)
	eout, err := sh.Command("/bin/sh", "-c", restore).SetTimeout(time.Second * 300).CombinedOutput()
	if err != nil {
		log.Fatalf("Restoration failed due to %s", eout)
	}
	cleanup := fmt.Sprintf("rm /tmp/%s.gz", m.Connection.BackupName)
	cout, cerr := sh.Command("/bin/sh", "-c", cleanup).SetTimeout(time.Second * 10).CombinedOutput()
	if cerr != nil {
		log.Fatalf("Failed to cleanup %s due to %s", m.Connection.BackupName, string(cout))
	}
}

func (m *MongoProvider) mongoBackupInsert(name string, time string, dbName string, path string, size int64) {
	coll := m.client.Database("backup").Collection("backups")
	_, err := coll.InsertOne(
		context.Background(),
		bson.D{
			{"name", name},
			{"time", time},
			{"database", dbName},
			{"path", path},
			{"size", size},
		})
	if err != nil {
		log.Fatal(err)
	}
	//var backup Backup
	//filter := result
	//coll.FindOne(context.TODO(), filter).Decode(&backup)
	log.Printf("The backup was %dMB\n", size)
}

func backupEncrypt(backupName string, password string) error {
	log.Printf("Encrypting %s\n", backupName)
	// TODO: Store encryption key in something like Vault
	encrypt := fmt.Sprintf("openssl aes-256-cbc -salt -pbkdf2 -in /tmp/%s.gz -out /tmp/%s.aes -k %s", backupName, backupName, password)
	_, eerr := sh.Command("/bin/sh", "-c", encrypt).SetTimeout(time.Second * 180).CombinedOutput()
	if eerr != nil {
		return eerr
	}
	cleanup := fmt.Sprintf("rm /tmp/%s.gz", backupName)
	_, cerr := sh.Command("/bin/sh", "-c", cleanup).SetTimeout(time.Second * 10).CombinedOutput()
	if cerr != nil {
		return cerr
	}
	return nil
}

func backupDecrypt(backupName string, password string) {
	log.Printf("Decrypting %s\n", backupName)
	// TODO: Retrieve decryption key in something like Vault
	decrypt := fmt.Sprintf("openssl aes-256-cbc -d -salt -pbkdf2 -in /tmp/%s.aes -out /tmp/%s.gz -k %s", backupName, backupName, password)
	out, err := sh.Command("/bin/sh", "-c", decrypt).SetTimeout(time.Second * 300).CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to decrypt because of %s", out)
	}
}
