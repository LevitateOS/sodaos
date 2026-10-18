import ts from '../lit-check/node_modules/typescript';

export interface Source {
  Path: string;
  Text: string;
}
export interface Row {
  path: string;
  name: string;
  line: number;
  score: number;
}

function functionBody(node: ts.Node): node is ts.FunctionLikeDeclaration & {body: ts.ConciseBody} {
  return ts.isFunctionLike(node) && 'body' in node && !!node.body;
}

function statementDecision(node: ts.Node): number {
  switch (node.kind) {
    case ts.SyntaxKind.IfStatement:
    case ts.SyntaxKind.ForStatement:
    case ts.SyntaxKind.ForInStatement:
    case ts.SyntaxKind.ForOfStatement:
    case ts.SyntaxKind.WhileStatement:
    case ts.SyntaxKind.DoStatement:
    case ts.SyntaxKind.CaseClause:
    case ts.SyntaxKind.CatchClause:
    case ts.SyntaxKind.ConditionalExpression:
      return 1;
    default:
      return 0;
  }
}

function operatorDecision(node: ts.Node): number {
  if (
    ts.isBinaryExpression(node) &&
    [
      ts.SyntaxKind.AmpersandAmpersandToken,
      ts.SyntaxKind.BarBarToken,
      ts.SyntaxKind.QuestionQuestionToken,
      ts.SyntaxKind.AmpersandAmpersandEqualsToken,
      ts.SyntaxKind.BarBarEqualsToken,
      ts.SyntaxKind.QuestionQuestionEqualsToken,
    ].includes(node.operatorToken.kind)
  )
    return 1;
  return 0;
}

function syntaxDecision(node: ts.Node): number {
  let score = 0;
  if ((ts.isParameter(node) || ts.isBindingElement(node)) && node.initializer) score++;
  if (
    (ts.isPropertyAccessExpression(node) || ts.isElementAccessExpression(node) || ts.isCallExpression(node)) &&
    node.questionDotToken
  )
    score++;
  return score;
}

function decisions(node: ts.Node, root: ts.Node): number {
  if (node !== root && functionBody(node)) return 0;
  let score = statementDecision(node) + operatorDecision(node) + syntaxDecision(node);
  ts.forEachChild(node, (child) => {
    score += decisions(child, root);
  });
  return score;
}

function label(node: ts.Node, file: ts.SourceFile): string {
  if ('name' in node && node.name) return (node.name as ts.Node).getText(file);
  if (ts.isConstructorDeclaration(node)) return 'constructor';
  if (ts.isVariableDeclaration(node.parent) || ts.isPropertyAssignment(node.parent))
    return node.parent.name.getText(file);
  return '<anonymous>';
}

export function measureTypeScript(files: Source[]): Row[] {
  const rows: Row[] = [];
  for (const file of files) {
    const tree = ts.createSourceFile(
      file.Path,
      file.Text,
      ts.ScriptTarget.Latest,
      true,
      file.Path.endsWith('.tsx') ? ts.ScriptKind.TSX : ts.ScriptKind.TS
    );
    const errors = (tree as ts.SourceFile & {parseDiagnostics: readonly ts.Diagnostic[]}).parseDiagnostics;
    if (errors.length) throw new Error(`${file.Path}: ${ts.flattenDiagnosticMessageText(errors[0]!.messageText, ' ')}`);
    const occurrences = new Map<string, number>();
    function visit(node: ts.Node, scope: string[]) {
      if (functionBody(node)) {
        const key = [...scope, label(node, tree)].join('::');
        const ordinal = (occurrences.get(key) ?? 0) + 1;
        occurrences.set(key, ordinal);
        const name = ordinal === 1 ? key : `${key}#${ordinal}`;
        rows.push({
          path: file.Path,
          name,
          line: tree.getLineAndCharacterOfPosition(node.getStart(tree)).line + 1,
          score: 1 + decisions(node, node),
        });
        ts.forEachChild(node, (child) => visit(child, [...scope, name]));
        return;
      }
      const nested = ts.isClassDeclaration(node) || ts.isClassExpression(node) || ts.isModuleDeclaration(node);
      ts.forEachChild(node, (child) => visit(child, nested ? [...scope, label(node, tree)] : scope));
    }
    visit(tree, []);
  }
  return rows;
}
