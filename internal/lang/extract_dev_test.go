package lang

import (
	"fmt"
	"testing"
)

// TestDevDump prints extraction output for one fixture per language.
// Run with: go test ./internal/lang/ -run TestDevDump -v
// It is a development aid; assertions live in extract_test.go.
func TestDevDump(t *testing.T) {
	fixtures := map[string]string{
		"go": `package auth

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

func NewServer() *Server { return &Server{} }

type Authenticator interface {
	Authenticate(token string) bool
}

const MaxRetries = 3

var DefaultTimeout = 30
`,
		"ts": `import { APIClient } from "./client";
import * as fs from "fs";
import React from "react";

export interface TokenStore {
	save(token: string): void;
}

export type UserID = string;

export enum Role {
	Admin,
	User,
}

/** AuthService handles login. */
export class AuthService extends BaseService implements TokenStore {
	api: APIClient;

	save(token: string): void {
		this.api.post("/token", token);
	}

	async login(user: string): Promise<void> {
		const client = new APIClient();
		client.post("/login", user);
	}
}

export function helper(): number { return 1; }

export const MAX_RETRIES = 3;
export const login = async (u: string) => { helper(); };
export let counter = 0;
`,
		"python": `"""Module doc."""
import os
import sys as system
from collections import OrderedDict
from mypkg.sub import Thing, helper


class AuthService(BaseService):
    """Auth service."""

    def login(self, user):
        """Login a user."""
        store = TokenStore()
        return store.save(user)


def top_level_func(a, b):
    helper(a)
    return os.path.join(a, str(b))


GLOBAL_VAR = 42
`,
		"rust": `use std::collections::HashMap;
use std::fmt::{self, Display};

/// A token store.
pub struct TokenStore {
    tokens: HashMap<String, String>,
}

pub enum Role {
    Admin,
    User,
}

pub trait Authenticator {
    fn authenticate(&self, token: &str) -> bool;
}

impl Display for TokenStore {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "store")
    }
}

impl TokenStore {
    /// Issue a token.
    pub fn issue(&self, user: &str) -> String {
        let m = HashMap::new();
        self.helper()
    }

    fn helper(&self) -> String { String::new() }
}

pub fn global_func(x: i32) -> i32 { x + 1 }

pub const MAX: i32 = 3;
pub static NAME: &str = "x";

mod inner {}
`,
		"java": `package com.example;

import java.util.List;
import com.example.BaseService;

/**
 * Auth service.
 */
public class AuthService extends BaseService implements Runnable {
    private final TokenStore store;

    public static final int MAX_RETRIES = 3;

    public AuthService(TokenStore store) {
        this.store = store;
    }

    public boolean login(String user) {
        store.save(user);
        List<String> l = new java.util.ArrayList<>();
        return true;
    }

    @Override
    public void run() { login("x"); }
}

interface Greeter { void greet(); }

enum Role { ADMIN, USER }
`,
		"kotlin": `package com.example

import kotlin.collections.List

/** Auth service. */
class AuthService(store: TokenStore) : BaseService(), Runnable {
    val maxRetries: Int = 3

    fun login(user: String): Boolean {
        val ok = store.save(user)
        println("login")
        return ok
    }

    override fun run() { login("x") }
}

object Singleton {
    fun helper() {}
}

fun topLevel(x: Int): Int = x + 1
`,
		"c": `#include <stdio.h>
#include "local.h"

/* A token store. */
typedef struct TokenStore {
    char tokens[64];
} TokenStore;

struct Server {
    int port;
};

enum Role { ADMIN, USER };

static const int MAX_RETRIES = 3;

int login(TokenStore *store, const char *user) {
    issue_token(store, user);
    printf("%s", user);
    return 0;
}

void helper(void) { login(NULL, "x"); }
`,
		"cpp": `#include <memory>
#include <vector>

namespace auth {

/// A token store.
class TokenStore : public Base {
public:
    int port;
    virtual bool save(const std::string& tok);
};

class Server {
public:
    Server();
    bool login(const std::string& user);
private:
    std::vector<int> cache_;
};

bool Server::login(const std::string& user) {
    auto p = std::make_unique<TokenStore>();
    p->save(user);
    return true;
}

using RoleID = int;

} // namespace auth
`,
	}

	for name, src := range fixtures {
		lg := Detect("." + map[string]string{
			"go": "go", "ts": "ts", "tsx": "tsx", "python": "py", "rust": "rs",
			"java": "java", "kotlin": "kt", "c": "c", "cpp": "cpp",
		}[name])
		if lg == nil {
			t.Fatalf("no language registered for %s", name)
		}
		res := lg.Extract([]byte(src))
		if !testing.Verbose() {
			continue
		}
		fmt.Printf("=== %s (%s) ===\n", name, lg.Name)
		for _, s := range res.Symbols {
			fmt.Printf("  SYM %-10s %-28s parent=%-12q sig=%.60s doc=%.30q\n",
				s.Kind, s.Display(), s.Parent, s.Signature, s.Doc)
		}
		for _, r := range res.Refs {
			fmt.Printf("  REF %-10s %-24q qual=%-10q in=%-20q @%d\n",
				r.Kind, r.Name, r.Qualifier, r.InSymbol, r.StartLine)
		}
		for _, i := range res.Imports {
			fmt.Printf("  IMP %-30q names=%v\n", i.Path, i.Names)
		}
	}
}
