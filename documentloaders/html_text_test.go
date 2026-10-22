package documentloaders

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/textsplitter"
)

func TestHTMLLoaderPreservesText(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		html string
		want string
	}{
		{"plain text", "<p>Visible text</p>", "Visible text"},
		{"ampersand", "<p>Research &amp; Development</p>", "Research & Development"},
		{"comparisons", "<p>1 &lt; 2 and 3 &gt; 2</p>", "1 < 2 and 3 > 2"},
		{"generic type", "<pre>vector&lt;int&gt; data;</pre>", "vector<int> data;"},
		{"quoted query", "<p>q=&quot;R&amp;D&quot;&amp;limit=2</p>", "q=\"R&D\"&limit=2"},
		{"literal allowed tags", "<code>&lt;em&gt;important&lt;/em&gt;</code>", "<em>important</em>"},
		{"literal script example", "<pre>&lt;script&gt;example()&lt;/script&gt;</pre>", "<script>example()</script>"},
		{"entity decoded once", "<code>&amp;lt;T&amp;gt; &amp;amp;</code>", "&lt;T&gt; &amp;"},
		{"numeric entity", "<p>&#38; &#x3c;T&#x3e;</p>", "& <T>"},
		{"Unicode", "<p>中文 &amp; 日本語 😀</p>", "中文 & 日本語 😀"},
		{"head excluded", "<html><head><title>Hidden</title><script>headCode()</script></head><body><p>Visible</p></body></html>", "Visible"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			docs, err := NewHTML(strings.NewReader(tt.html)).Load(context.Background())
			require.NoError(t, err)
			require.Len(t, docs, 1)
			require.Equal(t, tt.want, docs[0].PageContent)
			require.Empty(t, docs[0].Metadata)
		})
	}
}

func TestHTMLLoaderSanitizesMarkupBeforeExtractingText(t *testing.T) {
	t.Parallel()
	markup := `<body><script>scriptOnly()</script><style>.styleOnly{color:red}</style><p onclick="attrOnly()">Visible &amp; text</p><pre>&lt;script&gt;displayed()&lt;/script&gt;</pre></body>`
	docs, err := NewHTML(strings.NewReader(markup)).Load(context.Background())
	require.NoError(t, err)
	require.Len(t, docs, 1)
	require.Equal(t, "Visible & text<script>displayed()</script>", docs[0].PageContent)
}

func TestHTMLLoaderLoadAndSplitPreservesText(t *testing.T) {
	t.Parallel()
	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(1000),
		textsplitter.WithChunkOverlap(0),
	)
	docs, err := NewHTML(strings.NewReader("<pre>vector&lt;int&gt; values &amp; count;</pre>")).LoadAndSplit(context.Background(), splitter)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	require.Equal(t, "vector<int> values & count;", docs[0].PageContent)
	require.Empty(t, docs[0].Metadata)
}

func TestHTMLLoaderReturnsReaderError(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("read failed")
	for _, prefix := range []string{"", "<p>Partial text</p>"} {
		t.Run(prefix, func(t *testing.T) {
			t.Parallel()
			reader := io.MultiReader(strings.NewReader(prefix), iotest.ErrReader(wantErr))
			docs, err := NewHTML(reader).Load(context.Background())
			require.ErrorIs(t, err, wantErr)
			require.Nil(t, docs)
		})
	}
}
