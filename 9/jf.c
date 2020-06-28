#include <u.h>
#include <libc.h>
#include <bio.h>

#define MAXPATHLEN 4096
#define MAXKEYLEN 256
#define MAXNESTING 64

/* TODO: Paths are silently clipped if they get too long. */
Rune path[MAXPATHLEN];
int pend = 0;
int pends[MAXNESTING];
int nextpend = 0;
Rune key[MAXKEYLEN];

void
pathappendkey(Rune *s)
{
	if (nextpend == MAXNESTING-1)
		sysfatal("too much nesting");
	pends[nextpend++] = pend;
	if (pend == 1)
		pend += runesnprint(&path[pend], MAXPATHLEN-pend, "%S", s);
	else
		pend += runesnprint(&path[pend], MAXPATHLEN-pend, ".%S", s);
}

void
pathappendindex(int i)
{
	if (nextpend == MAXNESTING-1)
		sysfatal("too much nesting");
	pends[nextpend++] = pend;
	pend += runesnprint(&path[pend], MAXPATHLEN-pend, "[%d]", i);
}

void
pathbackup()
{
	if (nextpend == 0)
		sysfatal("unbalanced unnesting");
	pend = pends[--nextpend];
}

Biobuf bin;
Biobuf bout;

inline void
ignorespace(void)
{
	long r;
	for (;;) {
		r = Bgetrune(&bin);
		if (r == 0x20 || r == 0x0a || r == 0x0d || r == 0x09)
			continue;
		if (r != Beof)
			Bungetrune(&bin);
		return;
	}
}

inline void
expect(long r)
{
	long r1 = Bgetrune(&bin);
	if (r != r1)
		sysfatal("unexpected rune %C (%d), wanted %C (%d)", r1, r1, r, r);
}

void parsevalue(void);
void parsequoted(void);

/* Very similar to parse quoted string, but stores in a buffer. */
/* TODO: Too long keys are silently truncated. */
void
parsekey(Rune *s, int len)
{
	Rune *p = s;
	Rune *e = s + len;
	long r;
	expect('"');
	*p++ = '"';
	while (p < e) {
		r = Bgetrune(&bin);
		if (r == Beof)
			return;
		if (r == '\\') {
			*p++ = r;
			*p++ = Bgetrune(&bin);
			continue;
		}
		*p++ = r;
		if (r == '"') {
			*p = 0;
			return;
		}
	}
}

void
parseobject(void)
{
	Bprint(&bout, "%S	{}\n", path);
	expect('{');
	ignorespace();
	if (Bgetrune(&bin) == '}')
		return;
	else
		Bungetrune(&bin);
again:
	ignorespace();
	parsekey(key, MAXKEYLEN);
	ignorespace();
	expect(':');
	ignorespace();
	pathappendkey(key);
	parsevalue();
	pathbackup();
	ignorespace();
	long r = Bgetrune(&bin);
	if (r == '}')
		return;
	if (r == ',')
		goto again;
	sysfatal("unexpected rune after key-value pair: %C (%d)", r, r);
}

void
parsearray(void)
{
	Bprint(&bout, "%S	[]\n", path);
	expect('[');
	ignorespace();
	if (Bgetrune(&bin) == ']')
		return;
	else
		Bungetrune(&bin);
	int i = -1;
again:
	i++;
	ignorespace();
	pathappendindex(i);
	parsevalue();
	pathbackup();
	ignorespace();
	long r = Bgetrune(&bin);
	if (r == ']')
		return;
	if (r == ',')
		goto again;
	sysfatal("unexpected rune after array value: %C (%d)", r, r);
}

void
parsequoted(void)
{
	Bprint(&bout, "%S	\"", path);
	expect('"');
	long r;
	for (;;) {
		r = Bgetrune(&bin);
		if (r == Beof)
			return;
		if (r == '\\') {
			Bputrune(&bout, r);
			/* TODO: Could be EOF. */
			Bputrune(&bout, Bgetrune(&bin));
			continue;
		}
		Bputrune(&bout, r);
		if (r == '"') {
			Bputc(&bout, 0x0a);
			return;
		}
	}
}

void
parseunquoted(void)
{
	Bprint(&bout, "%S	", path);
	long r;
	for (;;) {
		r = Bgetrune(&bin);
		if (r == Beof) {
			return;
		}
		if (r == 0x20 || r == 0x0a || r == 0x0d || r == 0x09 || r == ':' || r == ',' || r== '[' || r == ']' || r == '{' || r == '}') {
			Bungetrune(&bin);
			Bputc(&bout, 0x0a);
			return;
		}
		Bputrune(&bout, r);
	}
}

void
parsevalue(void)
{
	ignorespace();
	long r = Bgetrune(&bin);
	Bungetrune(&bin);
	if (r == '{')
		parseobject();
	else if (r == '[')
		parsearray();
	else if (r == '"')
		parsequoted();
	else
		parseunquoted();
}

void
main(void)
{
	Binit(&bin, 0, OREAD);
	Binit(&bout, 1, OWRITE);
	pathappendkey(L"");
	parsevalue();
	pathbackup();
	ignorespace();
	if (pend != 0 || nextpend != 0)
		sysfatal("lingering element on stack: %S pend=%d nextpend=%d", path, pend, nextpend);
	Rune r = Bgetrune(&bin);
	if (r != Beof)
		sysfatal("trailing content after parsing value: %C (%d)", r, r);
	Bflush(&bout);
	exits(nil);
}
