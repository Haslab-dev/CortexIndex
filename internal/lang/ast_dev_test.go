package lang

import (
	"fmt"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/kotlin"
	tsgo "github.com/smacker/go-tree-sitter/typescript/typescript"
)

// TestASTDump prints raw tree shapes for grammar nodes we're unsure about.
// Run: go test ./internal/lang/ -run TestASTDump -v
func TestASTDump(t *testing.T) {
	dump := func(name string, lg *sitter.Language, src string) {
		if !testing.Verbose() {
			return
		}
		p := sitter.NewParser()
		p.SetLanguage(lg)
		tree := p.Parse(nil, []byte(src))
		fmt.Printf("=== %s ===\n%s\n", name, tree.RootNode().String())
		tree.Close()
	}
	dump("go-interface", golang.GetLanguage(), `package x
type A interface {
	DoThing() error
}
`)
	dump("ts-imports", tsgo.GetLanguage(), `import { APIClient, Helper as H } from "./client";
import * as fs from "fs";
import React from "react";
import type { X } from "./types";
export class A { field: number = 1; }
`)
	dump("kotlin", kotlin.GetLanguage(), `package com.example

import kotlin.collections.List

class AuthService(store: TokenStore) : BaseService(), Runnable {
    val maxRetries: Int = 3

    fun login(user: String): Boolean {
        return true
    }
}

fun topLevel(x: Int): Int = x + 1
`)
}
