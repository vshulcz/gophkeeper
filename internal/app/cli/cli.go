package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"gophkeeper/internal/domain/secret"
	"gophkeeper/internal/infra/client"
	"gophkeeper/internal/version"
	"io"
	"os"
	"strings"
	"time"

	appclient "gophkeeper/internal/app/client"

	"github.com/google/uuid"
)

// Service exposes client operations for CLI.
type Service interface {
	Register(ctx context.Context, username, password string) error
	Login(ctx context.Context, username, password string) error
	Add(ctx context.Context, input appclient.AddInput) (uuid.UUID, error)
	List(ctx context.Context, userPassword string) ([]appclient.ListItem, error)
	Get(ctx context.Context, id uuid.UUID, userPassword string) (appclient.PayloadEnvelope, error)
	Sync(ctx context.Context) (int, error)
}

// Deps provides external dependencies for CLI.
type Deps struct {
	NewService func(server string) Service
	ReadFile   func(path string) ([]byte, error)
	NowVersion func() (string, string)
}

// Run executes the CLI with provided dependencies and output writer.
func Run(args []string, deps Deps, out io.Writer) error {
	if len(args) < 1 {
		usage(out)
		return nil
	}
	cmd := args[0]
	switch cmd {
	case "register":
		return runRegister(args[1:], deps, out)
	case "login":
		return runLogin(args[1:], deps, out)
	case "add":
		return runAdd(args[1:], deps, out)
	case "list":
		return runList(args[1:], deps, out)
	case "get":
		return runGet(args[1:], deps, out)
	case "sync":
		return runSync(args[1:], deps, out)
	case "version":
		ver, date := deps.NowVersion()
		_, _ = fmt.Fprintf(out, "version: %s\nbuild date: %s\n", ver, date)
		return nil
	default:
		usage(out)
		return nil
	}
}

func usage(out io.Writer) {
	_, _ = fmt.Fprintln(out, "GophKeeper CLI")
	_, _ = fmt.Fprintln(out, "Usage: gophkeeper <command> [options]")
	_, _ = fmt.Fprintln(out, "Commands: register, login, add, list, get, sync, version")
}

func runRegister(args []string, deps Deps, out io.Writer) error {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	fs.SetOutput(out)
	server := fs.String("server", "http://localhost:8080", "server url")
	user := fs.String("user", "", "username")
	pass := fs.String("pass", "", "password")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := require(*user != "" && *pass != "", "user and pass required"); err != nil {
		return err
	}
	svc := deps.NewService(*server)
	if err := svc.Register(context.Background(), *user, *pass); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, "registered and logged in")
	return nil
}

func runLogin(args []string, deps Deps, out io.Writer) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(out)
	server := fs.String("server", "http://localhost:8080", "server url")
	user := fs.String("user", "", "username")
	pass := fs.String("pass", "", "password")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := require(*user != "" && *pass != "", "user and pass required"); err != nil {
		return err
	}
	svc := deps.NewService(*server)
	if err := svc.Login(context.Background(), *user, *pass); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(out, "logged in")
	return nil
}

func runAdd(args []string, deps Deps, out io.Writer) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(out)
	server := fs.String("server", "http://localhost:8080", "server url")
	itemType := fs.String("type", string(secret.TypeText), "item type")
	metaRaw := fs.String("meta", "", "comma-separated key=value meta")
	text := fs.String("text", "", "text data")
	login := fs.String("login", "", "login")
	password := fs.String("password", "", "password")
	file := fs.String("file", "", "binary file path")
	cardNumber := fs.String("card-number", "", "card number")
	cardExpiry := fs.String("card-expiry", "", "card expiry")
	cardHolder := fs.String("card-holder", "", "card holder")
	cardCVV := fs.String("card-cvv", "", "card cvv")
	userPass := fs.String("user-pass", "", "user password for encryption")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := require(*userPass != "", "user-pass required for encryption"); err != nil {
		return err
	}

	meta := parseKV(*metaRaw)
	typ, err := secret.ParseType(*itemType)
	if err != nil {
		return err
	}
	params := addParams{
		text:       *text,
		login:      *login,
		password:   *password,
		file:       *file,
		cardNumber: *cardNumber,
		cardExpiry: *cardExpiry,
		cardHolder: *cardHolder,
		cardCVV:    *cardCVV,
	}
	value, err := buildAddValue(typ, params, deps.ReadFile)
	if err != nil {
		return err
	}

	svc := deps.NewService(*server)
	id, err := svc.Add(context.Background(), appclient.AddInput{Value: value, Meta: meta, UserPassword: *userPass})
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "added item %s\n", id)
	return nil
}

type addParams struct {
	text       string
	login      string
	password   string
	file       string
	cardNumber string
	cardExpiry string
	cardHolder string
	cardCVV    string
}

func buildAddValue(typ secret.Type, params addParams, readFile func(string) ([]byte, error)) (secret.Value, error) {
	switch typ {
	case secret.TypeText:
		if err := require(params.text != "", "text required"); err != nil {
			return nil, err
		}
		return secret.Text{Text: params.text}, nil
	case secret.TypeLoginPassword:
		if err := require(params.login != "" && params.password != "", "login and password required"); err != nil {
			return nil, err
		}
		return secret.LoginPassword{Login: params.login, Password: params.password}, nil
	case secret.TypeBinary:
		if err := require(params.file != "", "file required"); err != nil {
			return nil, err
		}
		b, err := readFile(params.file)
		if err != nil {
			return nil, err
		}
		return secret.Binary{Base64: appclient.EncodePayload(b)}, nil
	case secret.TypeCard:
		if err := require(params.cardNumber != "" && params.cardExpiry != "" && params.cardHolder != "" && params.cardCVV != "", "card fields required"); err != nil {
			return nil, err
		}
		return secret.Card{
			Number: params.cardNumber,
			Expiry: params.cardExpiry,
			Holder: params.cardHolder,
			CVV:    params.cardCVV,
		}, nil
	default:
		return nil, errors.New("unknown type")
	}
}

func runList(args []string, deps Deps, out io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(out)
	userPass := fs.String("user-pass", "", "user password for decryption")
	if err := fs.Parse(args); err != nil {
		return err
	}

	svc := deps.NewService("http://localhost:8080")
	items, err := svc.List(context.Background(), *userPass)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(out, "no items")
		return nil
	}
	for _, item := range items {
		_, _ = fmt.Fprintf(out, "%s | %s | v=%d\n", item.ID, item.Type, item.Version)
		if item.Err != nil {
			_, _ = fmt.Fprintf(out, "  decrypt error: %v\n", item.Err)
			continue
		}
		if len(item.Meta) > 0 {
			_, _ = fmt.Fprintf(out, "  meta: %v\n", item.Meta)
		}
	}
	return nil
}

func runGet(args []string, deps Deps, out io.Writer) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	fs.SetOutput(out)
	id := fs.String("id", "", "item id")
	userPass := fs.String("user-pass", "", "user password for decryption")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := require(*id != "", "id required"); err != nil {
		return err
	}
	if err := require(*userPass != "", "user-pass required"); err != nil {
		return err
	}

	svc := deps.NewService("http://localhost:8080")
	itemID, err := uuid.Parse(*id)
	if err != nil {
		return err
	}
	resp, err := svc.Get(context.Background(), itemID, *userPass)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(out, "type: %s\n", resp.Type)
	_, _ = fmt.Fprintf(out, "meta: %v\n", resp.Meta)
	_, _ = fmt.Fprintf(out, "data: %v\n", resp.Data)
	return nil
}

func runSync(args []string, deps Deps, out io.Writer) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	fs.SetOutput(out)
	server := fs.String("server", "http://localhost:8080", "server url")
	if err := fs.Parse(args); err != nil {
		return err
	}

	svc := deps.NewService(*server)
	count, err := svc.Sync(context.Background())
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "synced %d items\n", count)
	return nil
}

func parseKV(raw string) map[string]string {
	res := map[string]string{}
	if raw == "" {
		return res
	}
	pairs := strings.Split(raw, ",")
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			continue
		}
		res[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return res
}

func require(ok bool, msg string) error {
	if !ok {
		return errors.New(msg)
	}
	return nil
}

// DefaultDeps returns production dependencies.
func DefaultDeps() Deps {
	return Deps{
		NewService: func(server string) Service {
			api := client.NewAPIClient(server)
			store, err := client.NewFileStore()
			if err != nil {
				return appclient.NewService(api, &noopStore{}, client.NewCryptoService(), time.Now)
			}
			crypto := client.NewCryptoService()
			return appclient.NewService(api, store, crypto, time.Now)
		},
		ReadFile: os.ReadFile,
		NowVersion: func() (string, string) {
			return version.Version, version.BuildDate
		},
	}
}

type noopStore struct{}

func (noopStore) Load(_ context.Context) (*appclient.StoreState, error) {
	return &appclient.StoreState{}, nil
}
func (noopStore) Save(_ context.Context, _ *appclient.StoreState) error { return nil }
