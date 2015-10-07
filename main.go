package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/thoj/go-ircevent"

	"github.com/Depado/markov-bot/configuration"
)

// PrefixLen is the number of words per Prefix defined as the key for the map.
const PrefixLen = 2

var db *bolt.DB
var open bool

// Open opens the database and puts the lock on it
// Unused yet
func Open() error {
	var err error
	_, filename, _, _ := runtime.Caller(0)
	dbfile := path.Join(path.Dir(filename), "data.db")
	config := &bolt.Options{Timeout: 1 * time.Second}
	db, err = bolt.Open(dbfile, 0600, config)
	if err != nil {
		log.Fatal(err)
	}
	open = true
	return nil
}

// Close closes the database
func Close() {
	open = false
	db.Close()
}

// Prefix is a Markov chain prefix of one or more words.
type Prefix []string

// String returns the Prefix as a string (for use as a map key).
func (p Prefix) String() string {
	return strings.Join(p, " ")
}

// Shift removes the first word from the Prefix and appends the given word.
func (p Prefix) Shift(word string) {
	copy(p, p[1:])
	p[len(p)-1] = word
}

// Chain contains a map ("chain") of prefixes to a list of suffixes.
// A prefix is a string of prefixLen words joined with spaces.
// A suffix is a single word. A prefix can have multiple suffixes.
type Chain struct {
	Nick  string
	Chain map[string][]string
}

// Save saves a chain to the database
func (c *Chain) Save() error {
	if !open {
		return fmt.Errorf("db must be opened before saving")
	}
	err := db.Update(func(tx *bolt.Tx) error {
		mBucket, err := tx.CreateBucketIfNotExists([]byte("markov"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		enc, err := c.Encode()
		if err != nil {
			return fmt.Errorf("Could not encode Chain : %s", err)
		}
		err = mBucket.Put([]byte(c.Nick), enc)
		return err
	})
	return err
}

// GetChain gets the markov chain associated to the nick
func GetChain(nick string) (*Chain, error) {
	if !open {
		return nil, fmt.Errorf("db must be opened before querying")
	}
	var c *Chain
	err := db.View(func(tx *bolt.Tx) error {
		var err error
		b := tx.Bucket([]byte("people"))
		k := []byte(nick)
		c, err = Decode(b.Get(k))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Encode encodes a chain to json.
func (c *Chain) Encode() ([]byte, error) {
	enc, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

// Decode decodes json to Chain
func Decode(data []byte) (*Chain, error) {
	var c *Chain
	err := json.Unmarshal(data, &c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// NewChain returns a new Chain with prefixes of prefixLen words.
func NewChain(nick string) *Chain {
	return &Chain{nick, make(map[string][]string)}
}

// Build reads text from the provided Reader and
// parses it into prefixes and suffixes that are stored in Chain.
func (c *Chain) Build(r io.Reader) {
	br := bufio.NewReader(r)
	p := make(Prefix, PrefixLen)
	for {
		var s string
		if _, err := fmt.Fscan(br, &s); err != nil {
			break
		}
		key := p.String()
		c.Chain[key] = append(c.Chain[key], s)
		p.Shift(s)
	}
}

// BuildFromString ...
func (c *Chain) BuildFromString(s string) {
	p := make(Prefix, PrefixLen)
	for _, v := range strings.Split(s, " ") {
		key := p.String()
		c.Chain[key] = append(c.Chain[key], v)
		p.Shift(v)
	}
}

// Generate returns a string of at most n words generated from Chain.
func (c *Chain) Generate() string {
	p := make(Prefix, PrefixLen)
	var words []string
	for {
		choices := c.Chain[p.String()]
		if len(choices) == 0 {
			break
		}
		next := choices[rand.Intn(len(choices))]
		words = append(words, next)
		p.Shift(next)
	}
	return strings.Join(words, " ")
}

// BuildFromFile builds a markov chain from a log file in a specific format
func BuildFromFile(nick, fname string) (*Chain, error) {
	full := NewChain(nick)
	file, err := os.Open(fname)
	if err != nil {
		return full, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		txt := strings.Split(scanner.Text(), " ")
		msg := strings.Join(txt[4:], " ")
		full.BuildFromString(msg)
	}
	if err = scanner.Err(); err != nil {
		return full, err
	}
	return full, nil
}

func main() {
	var err error
	rand.Seed(time.Now().UnixNano())

	configuration.Load("conf.yml")
	cnf := configuration.Config

	full, err := BuildFromFile("all", "history.log")
	if err != nil {
		log.Fatal(err)
	}

	ib := irc.IRC(cnf.BotName, cnf.BotName)
	if err = ib.Connect(cnf.Server); err != nil {
		log.Fatal(err)
	}

	ib.AddCallback("001", func(e *irc.Event) {
		ib.Join(cnf.Channel)
	})

	ib.AddCallback("PRIVMSG", func(e *irc.Event) {
		m := e.Message()
		if strings.HasPrefix(m, "!markov") {
			ib.Privmsg(cnf.Channel, full.Generate())
		}
	})

	ib.Loop()
}
