-- This Script is Part of the Prometheus Obfuscator by levno-710
--
-- enums.lua
--
-- This Script provides some enums used by the Obfuscator.

local Enums = {};

local chararray = require("prometheus.util").chararray;

Enums.LuaVersion = {
	LuaU = "LuaU" ,
	Lua51 = "Lua51",
}

Enums.Conventions = {
	[Enums.LuaVersion.Lua51] = {
		Keywords = {
			-- "goto" added by prometheus-lua-go: Lua 5.2+/LuaJIT/OTClient
			-- treat goto as a reserved word. Stock Lua 5.1 does not, but
			-- the obfuscator's target codebases do.
			"and", "break", "do", "else", "elseif",
			"end", "false", "for", "function", "goto", "if",
			"in", "local", "nil", "not", "or",
			"repeat", "return", "then", "true", "until", "while"
		},

		SymbolChars = chararray("+-*/%^#=~<>(){}[];:,."),
		MaxSymbolLength = 3,
		Symbols = {
			-- "::" added by prometheus-lua-go for goto-label support.
			"+", "-", "*", "/", "%", "^", "#",
			"==", "~=", "<=", ">=", "<", ">", "=",
			"(", ")", "{", "}", "[", "]",
			";", ":", "::", ",", ".", "..", "...",
		},

		IdentChars = chararray("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789"),
		NumberChars = chararray("0123456789"),
		HexNumberChars = chararray("0123456789abcdefABCDEF"),
		BinaryNumberChars = {"0", "1"},
		DecimalExponent = {"e", "E"},
		HexadecimalNums = {"x", "X"},
		BinaryNums = {"b", "B"},
		DecimalSeperators = false,

		EscapeSequences = {
			["a"] = "\a";
			["b"] = "\b";
			["f"] = "\f";
			["n"] = "\n";
			["r"] = "\r";
			["t"] = "\t";
			["v"] = "\v";
			["\\"] = "\\";
			["\""] = "\"";
			["\'"] = "\'";
		},
		NumericalEscapes = true,
		EscapeZIgnoreNextWhitespace = true,
		HexEscapes = true,
		UnicodeEscapes = true,
	},
	[Enums.LuaVersion.LuaU] = {
		Keywords = {
			"and", "break", "do", "else", "elseif", "continue",
			"end", "false", "for", "function", "if",
			"in", "local", "nil", "not", "or",
			"repeat", "return", "then", "true", "until", "while"
		},

		SymbolChars = chararray("+-*/%^#=~<>(){}[];:,."),
		MaxSymbolLength = 3,
		Symbols = {
			"+", "-", "*", "/", "%", "^", "#",
			"==", "~=", "<=", ">=", "<", ">", "=",
			"+=", "-=", "/=", "%=", "^=", "..=", "*=",
			"(", ")", "{", "}", "[", "]",
			";", ":", ",", ".", "..", "...",
			"::", "->", "?", "|", "&",
		},

		IdentChars = chararray("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_0123456789"),
		NumberChars = chararray("0123456789"),
		HexNumberChars = chararray("0123456789abcdefABCDEF"),
		BinaryNumberChars = {"0", "1"},
		DecimalExponent = {"e", "E"},
		HexadecimalNums = {"x", "X"},
		BinaryNums = {"b", "B"},
		DecimalSeperators = {"_"},

		EscapeSequences = {
			["a"] = "\a";
			["b"] = "\b";
			["f"] = "\f";
			["n"] = "\n";
			["r"] = "\r";
			["t"] = "\t";
			["v"] = "\v";
			["\\"] = "\\";
			["\""] = "\"";
			["\'"] = "\'";
		},
		NumericalEscapes = true,
		EscapeZIgnoreNextWhitespace = true,
		HexEscapes = true,
		UnicodeEscapes = true,
	},
}

return Enums;
