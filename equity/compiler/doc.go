/*
Package equity provides a compiler for Bytom's Equity contract language.

A contract is a means to lock some payment in the output of a
transaction. It contains a number of clauses, each describing a way to
unlock, or redeem, the payment in a subsequent transaction.  By
executing the statements in a clause, using contract arguments
supplied by the payer and clause arguments supplied by the redeemer,
nodes in a Bytom network can determine whether a proposed spend is
valid.

The language definition is in flux, but here's what's implemented as
of late Nov 2018.

  program = contract*

  contract = "contract" identifier "(" [params] ")" "locks" amount_identifier of asset_identifier "{" clause+ "}"

    The value(amount_identifier of asset_identifier) after "locks" is a name for
    the value locked by the contract. It must be unlocked or re-locked (with "unlock"
    or "lock") in every clause.

  clause = "clause" identifier "(" [params] ")" "{" statement+ "}"

  statement = verify | unlock | lock | define | assign | if/else

  verify = "verify" expr

    Verifies that boolean expression expr produces a true result.

  unlock = "unlock" expr "of" expr

    The first expr must be an amount, the second must be an asset.
    the value(expr "of" expr) must evaluate to the contract value.
    This unlocks that value for any use.

  lock = "lock" expr "of" expr "with" expr

    The first expr must be an amount, the second must be an asset.
    The later expr after "with" must be a program. This expression describe that
    the value(expr "of" expr) is unlocked and re-locks it with the new program immediately.

  define = "define" identifier : TypeName ["=" expr]

    Define a temporary variable "identifier" with type "TypeName". the identifier can be defined only
    or assigned with expr.

  assign = "assign" identifier "=" expr

    Assign a temporary variable "identifier" with expr. Please note that
    the "identifier" must be the defined variable with "define" expression.

  if = "if" expr "{" statement+ "}" [else "{" statement+ "}"]

    The check condition after "if" must be boolean expression. The if-else executes the statements
    inside the body of if-statement when condition expression is true, otherwise executes the statements
    inside the body of else-statement.

  params = param | params "," param

  param = identifier ":" type

    The identifier are individual parameter name. The identifier after the colon is their type.
    Available types are:

      Amount; Asset; Boolean; Hash; Integer; Program;
      PublicKey; Signature; String

  idlist = identifier | idlist "," identifier

  expr = unary_expr | binary_expr | call_expr | identifier | "(" expr ")" | literal

  unary_expr = unary_op expr

  binary_expr = expr binary_op expr

  call_expr = expr "(" [args] ")"

    If expr is the name of an Equity contract, then calling it (with
    the appropriate arguments) produces a program suitable for use
    in "lock" statements.

    Otherwise, expr should be one of these builtin functions:

      sha3(x)
        SHA3-256 hash of x.
      sha256(x)
        SHA-256 hash of x.
      size(x)
        Size in bytes of x.
      abs(x)
        Absolute value of x.
      min(x, y)
        The lesser of x and y.
      max(x, y)
        The greater of x and y.
      checkTxSig(pubkey, signature)
        Whether signature matches both the spending
        transaction and pubkey.
      concat(x, y)
        The concatenation of x and y.
      concatpush(x, y)
        The concatenation of x with the bytecode sequence
        needed to push y on the BVM stack.
      below(x)
        Whether the spending transaction is happening before
        blockHeight x.
      above(x)
        Whether the spending transaction is happening after
        blockHeight x.
      checkTxMultiSig([pubkey1, pubkey2, ...], [sig1, sig2, ...])
        Like checkTxSig, but for M-of-N signature checks.
        Every sig must match both the spending transaction and
        one of the pubkeys. There may be more pubkeys than
        sigs, but they are only checked left-to-right so must
        be supplied in the same order as the sigs. The square
        brackets here are literal and must appear as shown.

  unary_op = "-" | "~"

  binary_op = ">" | "<" | ">=" | "<=" | "==" | "!=" | "^" | "|" |
        "+" | "-" | "&" | "<<" | ">>" | "%" | "*" | "/"

  args = expr | args "," expr

  literal = int_literal | str_literal | hex_literal

*/
package compiler
