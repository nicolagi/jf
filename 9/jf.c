#include <u.h>
#include <libc.h>
#include <bio.h>

Rune *path;
int pathcap = 1;

int *pends;
int pend = 0;
int pendscap = 1;
int nextpend = 0;

/* TODO: Keys are silently clipped if they get too long. */
#define MAXKEYLEN 64
Rune key[MAXKEYLEN];

/* Adds an object key if not nil, otherwise an array index. */
void
pathappend(Rune *s, int i)
{
	if (nextpend == pendscap) {
		int *newpends = malloc(2*pendscap*sizeof(int));
		memcpy(newpends, pends, pendscap*sizeof(int));
		free(pends);
		pends = newpends;
		pendscap *= 2;
	}
	pends[nextpend++] = pend;
	int n;
retry:
	if (s != nil)
		n = runesnprint(&path[pend], pathcap-pend, pend == 1 ? "%S" : ".%S", s);
	else
		n = runesnprint(&path[pend], pathcap-pend, "[%d]", i);
	pend += n;
	if (pend == pathcap-1) {
		Rune *newpath = malloc(2*pathcap*sizeof(Rune));
		memcpy(newpath, path, pend*sizeof(Rune));
		free(path);
		path = newpath;
		pathcap *= 2;
		pend -= n;
		goto retry;
	}
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
	pathappend(key, 0);
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
	pathappend(nil, i);
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
	path = malloc(sizeof(Rune));
	pends = malloc(sizeof(int));
	Binit(&bin, 0, OREAD);
	Binit(&bout, 1, OWRITE);
	pathappend(L"", 0);
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
