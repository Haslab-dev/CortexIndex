package lang

import (
	"strings"
	"testing"
)

// findSym returns the symbol with the given display name.
func findSym(t *testing.T, res Result, display string) Symbol {
	t.Helper()
	for _, s := range res.Symbols {
		if s.Display() == display {
			return s
		}
	}
	t.Fatalf("symbol %q not found; have %v", display, symNames(res))
	return Symbol{}
}

func symNames(res Result) []string {
	var out []string
	for _, s := range res.Symbols {
		out = append(out, s.Display()+"("+string(s.Kind)+")")
	}
	return out
}

func hasRef(res Result, kind RefKind, name string) bool {
	for _, r := range res.Refs {
		if r.Kind == kind && r.Name == name {
			return true
		}
	}
	return false
}

func hasImport(res Result, path string) bool {
	for _, i := range res.Imports {
		if i.Path == path {
			return true
		}
	}
	return false
}

func runExtract(t *testing.T, ext, src string) Result {
	t.Helper()
	lg := Detect("file" + ext)
	if lg == nil {
		t.Fatalf("no language for extension %q", ext)
	}
	return lg.Extract([]byte(src))
}

func TestExtractGo(t *testing.T) {
	res := runExtract(t, ".go", `package auth

import (
	"fmt"
	str "strings"
)

// Server serves auth requests.
type Server struct {
	store TokenStore
}

// Login authenticates a user.
func (s *Server) Login(user string) error {
	tok, err := s.store.Issue(user)
	if err != nil {
		return fmt.Errorf("issue: %w", err)
	}
	str.ToUpper(tok)
	return nil
}

type Authenticator interface {
	Authenticate(token string) bool
}

const MaxRetries = 3
`)
	s := findSym(t, res, "Server.Login")
	if s.Kind != KindMethod || s.Parent != "Server" {
		t.Errorf("Login kind/parent = %s/%q", s.Kind, s.Parent)
	}
	if !strings.Contains(s.Signature, "func (s *Server) Login(user string) error") {
		t.Errorf("Login signature = %q", s.Signature)
	}
	if s.Doc != "Login authenticates a user." {
		t.Errorf("Login doc = %q", s.Doc)
	}
	if sl := findSym(t, res, "Server"); sl.Kind != KindStruct {
		t.Errorf("Server kind = %s", sl.Kind)
	}
	if sl := findSym(t, res, "Authenticator.Authenticate"); sl.Kind != KindMethod {
		t.Errorf("interface method kind = %s", sl.Kind)
	}
	if sl := findSym(t, res, "MaxRetries"); sl.Kind != KindConstant {
		t.Errorf("MaxRetries kind = %s", sl.Kind)
	}
	if !hasRef(res, RefCall, "Issue") || !hasRef(res, RefCall, "ToUpper") {
		t.Errorf("calls missing: %v", res.Refs)
	}
	if !hasImport(res, "fmt") || !hasImport(res, "strings") {
		t.Errorf("imports missing: %v", res.Imports)
	}
	// locations are 1-based lines
	if s.StartLine < 1 || s.EndLine < s.StartLine {
		t.Errorf("bad range for Login: %d-%d", s.StartLine, s.EndLine)
	}
}

func TestExtractTypeScript(t *testing.T) {
	res := runExtract(t, ".ts", `import { APIClient } from "./client";
import * as fs from "fs";

export class AuthService extends BaseService implements TokenStore {
	api: APIClient = new APIClient();

	async login(user: string): Promise<void> {
		this.api.post("/login", user);
	}
}

export interface TokenStore {
	save(token: string): void;
}

export const MAX_RETRIES = 3;
`)
	s := findSym(t, res, "AuthService.login")
	if s.Kind != KindMethod || s.Parent != "AuthService" {
		t.Errorf("login kind/parent = %s/%q", s.Kind, s.Parent)
	}
	if sl := findSym(t, res, "AuthService.api"); sl.Kind != KindProperty {
		t.Errorf("api kind = %s", sl.Kind)
	}
	if findSym(t, res, "TokenStore").Kind != KindInterface {
		t.Errorf("TokenStore should be interface")
	}
	if !hasRef(res, RefExtends, "BaseService") || !hasRef(res, RefImplements, "TokenStore") {
		t.Errorf("inheritance missing: %v", res.Refs)
	}
	if !hasRef(res, RefCall, "post") {
		t.Errorf("call missing")
	}
	if !hasImport(res, "./client") || !hasImport(res, "fs") {
		t.Errorf("imports missing: %v", res.Imports)
	}
}

func TestExtractPython(t *testing.T) {
	res := runExtract(t, ".py", `import os
from collections import OrderedDict


class AuthService(BaseService):
    """Auth service."""

    def login(self, user):
        """Login a user."""
        return store.save(user)
`)
	s := findSym(t, res, "AuthService.login")
	if s.Kind != KindFunction || s.Parent != "AuthService" {
		t.Errorf("login kind/parent = %s/%q", s.Kind, s.Parent)
	}
	if s.Doc != "Login a user." {
		t.Errorf("docstring = %q", s.Doc)
	}
	if !hasRef(res, RefExtends, "BaseService") || !hasRef(res, RefCall, "save") {
		t.Errorf("refs missing: %v", res.Refs)
	}
	if !hasImport(res, "os") || !hasImport(res, "collections") {
		t.Errorf("imports missing: %v", res.Imports)
	}
}

func TestExtractRust(t *testing.T) {
	res := runExtract(t, ".rs", `use std::collections::HashMap;

pub struct TokenStore {
	tokens: HashMap<String, String>,
}

pub trait Authenticator {
	fn authenticate(&self, token: &str) -> bool;
}

impl TokenStore {
	/// Issue a token.
	pub fn issue(&self, user: &str) -> String {
		let m = HashMap::new();
		self.helper()
	}

	fn helper(&self) -> String { String::new() }
}
`)
	s := findSym(t, res, "TokenStore.issue")
	if s.Kind != KindFunction || s.Parent != "TokenStore" {
		t.Errorf("issue kind/parent = %s/%q", s.Kind, s.Parent)
	}
	if s.Doc != "Issue a token." {
		t.Errorf("doc = %q", s.Doc)
	}
	if findSym(t, res, "Authenticator").Kind != KindTrait {
		t.Errorf("trait kind wrong")
	}
	if !hasRef(res, RefCall, "helper") || !hasRef(res, RefCall, "new") {
		t.Errorf("calls missing: %v", res.Refs)
	}
	if !hasImport(res, "std::collections::HashMap") {
		t.Errorf("import missing: %v", res.Imports)
	}
}

func TestExtractJava(t *testing.T) {
	res := runExtract(t, ".java", `package com.example;

import java.util.List;

public class AuthService extends BaseService implements Runnable {
	public boolean login(String user) {
		store.save(user);
		return true;
	}

	@Override
	public void run() { login("x"); }
}
`)
	s := findSym(t, res, "AuthService.login")
	if s.Kind != KindMethod || s.Parent != "AuthService" {
		t.Errorf("login kind/parent = %s/%q", s.Kind, s.Parent)
	}
	if !hasRef(res, RefExtends, "BaseService") || !hasRef(res, RefImplements, "Runnable") {
		t.Errorf("inheritance missing: %v", res.Refs)
	}
	if !hasRef(res, RefCall, "login") {
		t.Errorf("self-call missing")
	}
	if !hasImport(res, "java.util.List") {
		t.Errorf("import missing")
	}
}

func TestExtractKotlin(t *testing.T) {
	res := runExtract(t, ".kt", `package com.example

import kotlin.collections.List

class AuthService(store: TokenStore) : BaseService() {
	val maxRetries: Int = 3

	fun login(user: String): Boolean {
		val ok = store.save(user)
		return ok
	}
}
`)
	if s := findSym(t, res, "AuthService.login"); s.Parent != "AuthService" {
		t.Errorf("login parent = %q", s.Parent)
	}
	if s := findSym(t, res, "AuthService.maxRetries"); s.Kind != KindProperty {
		t.Errorf("maxRetries kind = %s", s.Kind)
	}
	// local val inside login must not be indexed
	for _, s := range res.Symbols {
		if s.Name == "ok" {
			t.Errorf("local variable leaked: %v", res.Symbols)
		}
	}
	if !hasRef(res, RefExtends, "BaseService") || !hasRef(res, RefCall, "save") {
		t.Errorf("refs missing: %v", res.Refs)
	}
}

func TestExtractCAndCpp(t *testing.T) {
	res := runExtract(t, ".c", `#include <stdio.h>

typedef struct TokenStore {
	char tokens[64];
} TokenStore;

int login(TokenStore *store, const char *user) {
	issue_token(store, user);
	return 0;
}
`)
	if s := findSym(t, res, "login"); s.Kind != KindFunction {
		t.Errorf("login kind = %s", s.Kind)
	}
	if !hasImport(res, "stdio.h") {
		t.Errorf("include missing")
	}
	if !hasRef(res, RefCall, "issue_token") {
		t.Errorf("call missing")
	}

	cpp := runExtract(t, ".cpp", `class Server {
public:
	bool save(const char* u);
};

bool Server::save(const char* u) {
	return do_save(u);
}
`)
	s := findSym(t, cpp, "Server.save")
	if s.Parent != "Server" {
		t.Errorf("out-of-line method parent = %q", s.Parent)
	}
	if !strings.Contains(s.Signature, "Server::save") {
		t.Errorf("signature lost qualifier: %q", s.Signature)
	}
}

func TestNameParts(t *testing.T) {
	cases := map[string][]string{
		"AuthService":     {"auth", "service"},
		"getUserByID":     {"get", "user", "by", "id"},
		"HTTPServer":      {"http", "server"},
		"token_store":     {"token", "store"},
		"com.example.Foo": {"com", "example", "foo"},
	}
	for in, want := range cases {
		got := NameParts(in)
		if len(got) != len(want) {
			t.Errorf("NameParts(%q) = %v, want %v", in, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("NameParts(%q)[%d] = %q, want %q", in, i, got[i], want[i])
			}
		}
	}
}

func TestDetectTextExts(t *testing.T) {
	if Detect("README.md") != nil {
		t.Errorf("markdown must not be a parseable language")
	}
	if !IsTextExt("docs/notes.md") || !IsTextExt("config.yaml") {
		t.Errorf("text extensions not detected")
	}
	if IsTextExt("main.go") {
		t.Errorf("go must not be text-only")
	}
}
