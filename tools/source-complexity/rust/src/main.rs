use quote::ToTokens;
use serde::{Deserialize, Serialize};
use std::io;
use syn::{
    punctuated::Punctuated,
    spanned::Spanned,
    visit::{self, Visit},
    Attribute, Meta, Token,
};

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
struct Source {
    path: String,
    text: String,
}
#[derive(Serialize)]
struct ResultRow {
    path: String,
    name: String,
    line: usize,
    score: usize,
}

// Exclude a cfg only when it provably requires test. Unknown feature/target
// predicates remain included: this is a source gate, not a build-configuration gate.
fn requires_test(meta: &Meta) -> bool {
    match meta {
        Meta::Path(path) => path.is_ident("test"),
        Meta::List(list) => {
            let Ok(items) = list.parse_args_with(Punctuated::<Meta, Token![,]>::parse_terminated)
            else {
                return false;
            };
            if list.path.is_ident("all") {
                items.iter().any(requires_test)
            } else if list.path.is_ident("any") {
                !items.is_empty() && items.iter().all(requires_test)
            } else {
                false
            }
        }
        _ => false,
    }
}
fn test_only(attrs: &[Attribute]) -> bool {
    attrs.iter().any(|a| {
        a.path().segments.last().is_some_and(|s| s.ident == "test")
            || (a.path().is_ident("cfg") && a.parse_args::<Meta>().is_ok_and(|m| requires_test(&m)))
    })
}

// syn stores expression attributes on each concrete AST variant.
fn expr_test_only(expr: &syn::Expr) -> bool {
    let attrs: &[Attribute] = match expr {
        syn::Expr::Array(syn::ExprArray { attrs, .. })
        | syn::Expr::Assign(syn::ExprAssign { attrs, .. })
        | syn::Expr::Async(syn::ExprAsync { attrs, .. })
        | syn::Expr::Await(syn::ExprAwait { attrs, .. })
        | syn::Expr::Binary(syn::ExprBinary { attrs, .. })
        | syn::Expr::Block(syn::ExprBlock { attrs, .. })
        | syn::Expr::Break(syn::ExprBreak { attrs, .. })
        | syn::Expr::Call(syn::ExprCall { attrs, .. })
        | syn::Expr::Cast(syn::ExprCast { attrs, .. })
        | syn::Expr::Closure(syn::ExprClosure { attrs, .. })
        | syn::Expr::Const(syn::ExprConst { attrs, .. })
        | syn::Expr::Continue(syn::ExprContinue { attrs, .. })
        | syn::Expr::Field(syn::ExprField { attrs, .. })
        | syn::Expr::ForLoop(syn::ExprForLoop { attrs, .. })
        | syn::Expr::Group(syn::ExprGroup { attrs, .. })
        | syn::Expr::If(syn::ExprIf { attrs, .. })
        | syn::Expr::Index(syn::ExprIndex { attrs, .. })
        | syn::Expr::Infer(syn::ExprInfer { attrs, .. })
        | syn::Expr::Let(syn::ExprLet { attrs, .. })
        | syn::Expr::Lit(syn::ExprLit { attrs, .. })
        | syn::Expr::Loop(syn::ExprLoop { attrs, .. })
        | syn::Expr::Macro(syn::ExprMacro { attrs, .. })
        | syn::Expr::Match(syn::ExprMatch { attrs, .. })
        | syn::Expr::MethodCall(syn::ExprMethodCall { attrs, .. })
        | syn::Expr::Paren(syn::ExprParen { attrs, .. })
        | syn::Expr::Path(syn::ExprPath { attrs, .. })
        | syn::Expr::Range(syn::ExprRange { attrs, .. })
        | syn::Expr::RawAddr(syn::ExprRawAddr { attrs, .. })
        | syn::Expr::Reference(syn::ExprReference { attrs, .. })
        | syn::Expr::Repeat(syn::ExprRepeat { attrs, .. })
        | syn::Expr::Return(syn::ExprReturn { attrs, .. })
        | syn::Expr::Struct(syn::ExprStruct { attrs, .. })
        | syn::Expr::Try(syn::ExprTry { attrs, .. })
        | syn::Expr::TryBlock(syn::ExprTryBlock { attrs, .. })
        | syn::Expr::Tuple(syn::ExprTuple { attrs, .. })
        | syn::Expr::Unary(syn::ExprUnary { attrs, .. })
        | syn::Expr::Unsafe(syn::ExprUnsafe { attrs, .. })
        | syn::Expr::While(syn::ExprWhile { attrs, .. })
        | syn::Expr::Yield(syn::ExprYield { attrs, .. }) => attrs,
        _ => &[],
    };
    test_only(attrs)
}

struct Counter(usize);
impl<'ast> Visit<'ast> for Counter {
    fn visit_item(&mut self, _: &'ast syn::Item) {} // Nested functions are measured separately.
    fn visit_expr(&mut self, expr: &'ast syn::Expr) {
        if expr_test_only(expr) {
            return;
        }
        self.0 += match expr {
            syn::Expr::If(_) | syn::Expr::ForLoop(_) | syn::Expr::While(_) | syn::Expr::Try(_) => 1,
            syn::Expr::Binary(b) if matches!(b.op, syn::BinOp::And(_) | syn::BinOp::Or(_)) => 1,
            syn::Expr::Match(m) => m
                .arms
                .iter()
                .filter(|a| !test_only(&a.attrs))
                .map(|a| {
                    let patterns = match &a.pat {
                        syn::Pat::Wild(_) => 0,
                        syn::Pat::Or(p) => p.cases.len(),
                        _ => 1,
                    };
                    patterns + usize::from(a.guard.is_some())
                })
                .sum(),
            _ => 0,
        };
        visit::visit_expr(self, expr);
    }
    fn visit_arm(&mut self, arm: &'ast syn::Arm) {
        if !test_only(&arm.attrs) {
            visit::visit_arm(self, arm);
        }
    }
    fn visit_stmt(&mut self, stmt: &'ast syn::Stmt) {
        let attrs = match stmt {
            syn::Stmt::Local(local) => &local.attrs[..],
            syn::Stmt::Macro(mac) => &mac.attrs[..],
            _ => &[],
        };
        if !test_only(attrs) {
            visit::visit_stmt(self, stmt);
        }
    }
}

struct Functions {
    path: String,
    scope: Vec<String>,
    rows: Vec<ResultRow>,
}
impl Functions {
    fn function(&mut self, sig: &syn::Signature, block: &syn::Block, attrs: &[Attribute]) {
        if test_only(attrs) {
            return;
        }
        self.scope.push(sig.ident.to_string());
        let mut count = Counter(1);
        count.visit_block(block);
        self.rows.push(ResultRow {
            path: self.path.clone(),
            name: self.scope.join("::"),
            line: sig.span().start().line,
            score: count.0,
        });
        self.visit_block(block);
        self.scope.pop();
    }
}
impl<'ast> Visit<'ast> for Functions {
    fn visit_expr(&mut self, expr: &'ast syn::Expr) {
        if !expr_test_only(expr) {
            visit::visit_expr(self, expr);
        }
    }
    fn visit_arm(&mut self, arm: &'ast syn::Arm) {
        if !test_only(&arm.attrs) {
            visit::visit_arm(self, arm);
        }
    }
    fn visit_item_mod(&mut self, item: &'ast syn::ItemMod) {
        if test_only(&item.attrs) {
            return;
        }
        self.scope.push(item.ident.to_string());
        visit::visit_item_mod(self, item);
        self.scope.pop();
    }
    fn visit_item_impl(&mut self, item: &'ast syn::ItemImpl) {
        if test_only(&item.attrs) {
            return;
        }
        let mut name = item.self_ty.to_token_stream().to_string();
        if let Some((_, trait_path, _)) = &item.trait_ {
            name = format!("{} for {name}", trait_path.to_token_stream());
        }
        self.scope.push(name);
        visit::visit_item_impl(self, item);
        self.scope.pop();
    }
    fn visit_item_trait(&mut self, item: &'ast syn::ItemTrait) {
        if test_only(&item.attrs) {
            return;
        }
        self.scope.push(item.ident.to_string());
        visit::visit_item_trait(self, item);
        self.scope.pop();
    }
    fn visit_item_fn(&mut self, item: &'ast syn::ItemFn) {
        self.function(&item.sig, &item.block, &item.attrs);
    }
    fn visit_impl_item_fn(&mut self, item: &'ast syn::ImplItemFn) {
        self.function(&item.sig, &item.block, &item.attrs);
    }
    fn visit_trait_item_fn(&mut self, item: &'ast syn::TraitItemFn) {
        if let Some(block) = &item.default {
            self.function(&item.sig, block, &item.attrs);
        }
    }
}
fn main() -> Result<(), Box<dyn std::error::Error>> {
    let files: Vec<Source> = serde_json::from_reader(io::stdin())?;
    let mut out = Vec::new();
    for file in files {
        let tree = syn::parse_file(&file.text).map_err(|e| format!("{}: {e}", file.path))?;
        if test_only(&tree.attrs) {
            continue;
        }
        let mut functions = Functions {
            path: file.path,
            scope: Vec::new(),
            rows: Vec::new(),
        };
        functions.visit_file(&tree);
        out.extend(functions.rows);
    }
    serde_json::to_writer(io::stdout(), &out)?;
    Ok(())
}
