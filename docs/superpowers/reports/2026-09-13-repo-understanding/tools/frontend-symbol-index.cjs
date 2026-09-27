// Syntax inventory only. No call-target resolution or human-read claim.
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const [root, output, tsPath] = process.argv.slice(2);
const ts = require(tsPath);
const files = JSON.parse(fs.readFileSync(path.join(output,'file-inventory.json'),'utf8')).map(x=>x.path).filter(p => /\.(tsx?|jsx?)$/.test(p));
const functions = [], imports = [], errors = [];
for (const rel of files) {
  const raw = fs.readFileSync(path.join(root, rel), 'utf8');
  const source = ts.createSourceFile(rel, raw, ts.ScriptTarget.Latest, true);
  for (const e of source.parseDiagnostics) errors.push({path:rel,start:e.start,message:ts.flattenDiagnosticMessageText(e.messageText,' ')});
  const line = p => source.getLineAndCharacterOfPosition(p).line + 1;
  function visit(node) {
    if (ts.isImportDeclaration(node)) imports.push({path:rel,line:line(node.getStart(source)),module:node.moduleSpecifier.text});
    if (ts.isFunctionDeclaration(node) || ts.isFunctionExpression(node) || ts.isArrowFunction(node) || ts.isMethodDeclaration(node) || ts.isConstructorDeclaration(node) || ts.isGetAccessor(node) || ts.isSetAccessor(node)) {
      let name = node.name?.getText(source);
      if (!name && ts.isVariableDeclaration(node.parent)) name = node.parent.name.getText(source);
      if (!name && ts.isPropertyAssignment(node.parent)) name = node.parent.name.getText(source);
      name ||= ts.isConstructorDeclaration(node) ? 'constructor' : '<anonymous>';
      functions.push({path:rel,name,kind:ts.SyntaxKind[node.kind],start_line:line(node.getStart(source)),end_line:line(node.end),source_sha256:crypto.createHash('sha256').update(node.getText(source)).digest('hex'),resolution:'syntax-only'});
    }
    ts.forEachChild(node, visit);
  }
  visit(source);
}
for (const [name,data] of [['frontend-functions',functions],['frontend-imports',imports],['frontend-parser-errors',errors]]) fs.writeFileSync(path.join(output,name+'.json'),JSON.stringify(data,null,2)+'\n');
console.log(JSON.stringify({files:files.length,functions:functions.length,imports:imports.length,parseErrors:errors.length,typescript:ts.version}));
