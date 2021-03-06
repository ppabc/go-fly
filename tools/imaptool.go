package tools

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

//全局变量
//imap服务地址,邮箱,密码
var (
	server, email, password string
	folders                 map[int]string
)

func testmain() {

	//获取参数中的数据
	flag.StringVar(&server, "server", "", "imap服务地址(包含端口)")
	flag.StringVar(&email, "email", "", "邮箱名")
	flag.StringVar(&password, "password", "", "密码")
	flag.Parse()
	if flag.NFlag() < 3 {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if server == "" || email == "" || password == "" {
		log.Fatal("服务器地址,用户名,密码,参数必填")
	}
	log.Println("正在连接服务器...")
	//支持加密和非加密端口
	if !strings.Contains(server, ":") {
		log.Fatal("服务器地址端口号错误:", server)
	}
	serverSlice := strings.Split(server, ":")
	uri := serverSlice[0]
	port, _ := strconv.Atoi(serverSlice[1])
	if port != 993 && port != 143 {
		log.Fatal("服务器地址端口号错误:", port)
	}
	// 连接到服务器
	var c *client.Client
	var err error
	if port == 993 {
		c, err = client.DialTLS(fmt.Sprintf("%s:%d", uri, port), nil)
	} else {
		c, err = client.Dial(fmt.Sprintf("%s:%d", uri, port))
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Println("连接成功")

	// 不要忘了退出
	defer c.Logout()

	// 登陆
	if err := c.Login(email, password); err != nil {
		log.Fatal(err)
	}
	log.Println("成功登陆")

	// 列邮箱
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()
	// 存储邮件夹
	folders = make(map[int]string)
	i := 1
	for m := range mailboxes {
		log.Println("* ", i, m.Name)
		folders[i] = m.Name
		i++
	}
	log.Println("输入邮件夹序号:")
	inLine := readLineFromInput()
	folderNum, _ := strconv.Atoi(inLine)
	currentFolder := folders[folderNum]

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	// Select 邮件夹
	mbox, err := c.Select(currentFolder, false)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s的邮件个数:%d \r\n", currentFolder, mbox.Messages)

	// 获取最新的信
	log.Println("读取最新的几封信(all全部):")
	inLine = readLineFromInput()
	var maxNum uint32
	if inLine == "all" {
		maxNum = mbox.Messages
	} else {
		tempNum, _ := strconv.Atoi(inLine)
		maxNum = uint32(tempNum)
	}

	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages >= maxNum {
		// 我们在这使用无符号整型, 这是再获取from的id
		from = mbox.Messages - maxNum + 1
	} else {
		log.Fatal("超出了邮件封数!")
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	log.Printf("最新的 %d 封信:", maxNum)
	for msg := range messages {
		log.Printf("* %d:%s\n", to, msg.Envelope.Subject)
		to--
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}
	for {
		log.Println("读取第几封信信体:")

		inLine = readLineFromInput()
		mesIndex, _ := strconv.Atoi(inLine)
		mesSeqSet := new(imap.SeqSet)
		mesSeqSet.AddNum(uint32(mesIndex))
		//获取整封信体
		var section imap.BodySectionName
		items := []imap.FetchItem{section.FetchItem()}

		bodyMsg := make(chan *imap.Message, 1)
		go func() {
			if err := c.Fetch(mesSeqSet, items, bodyMsg); err != nil {
				log.Fatal(err)
			}
		}()
		msg := <-bodyMsg
		if msg == nil {
			log.Fatal("服务器没有返回信息")
		}
		r := msg.GetBody(&section)
		if r == nil {
			log.Fatal("服务器没有返回 message body")
		}
		mr, err := mail.CreateReader(r)
		if err != nil {
			log.Fatal(err)
		}

		//打印信息
		header := mr.Header
		if date, err := header.Date(); err == nil {
			log.Println("Date:", date)
		}
		if from, err := header.AddressList("From"); err == nil {
			log.Println("From:", from)
		}
		if to, err := header.AddressList("To"); err == nil {
			log.Println("To:", to)
		}
		if subject, err := header.Subject(); err == nil {
			log.Println("Subject:", subject)
		}
		// Process each message's part
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				log.Fatal(err)
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				// This is the message's text (can be plain-text or HTML)
				b, _ := ioutil.ReadAll(p.Body)
				log.Printf("Got text: %s\n", string(b))
			case *mail.AttachmentHeader:
				// This is an attachment
				filename, _ := h.Filename()
				log.Printf("Got attachment: %s\n", filename)
			}
		}
		log.Printf("读取第 %d 封信, 结束!\n", mesIndex)
	}
}

//从输入中读取一行
func readLineFromInput() string {
	str := ""
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		str = scanner.Text() // Println will add back the final '\n'
		break
	}
	return str
}
