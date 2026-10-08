package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckAllFormsOrdered(t *testing.T) {
	s := New(nil, nil)
	goVerb := Verb{
		Base: "go",
		Past: map[string][]string{
			"gb": {
				"went",
			},
		},
		Participle: map[string][]string{
			"gb": {
				"gone",
			},
		},
	}
	cases := []struct {
		in   string
		want bool
	}{
		{
			in:   "go went gone",
			want: true,
		},
		{
			in:   "to go went gone",
			want: true,
		}, // optional "to"
		{
			in:   "GO  went / gone",
			want: true,
		}, // mixed separators, case
		{
			in:   "go-went-gone",
			want: true,
		}, // hyphen separator
		{
			in:   "go - went - gone",
			want: true,
		},
		{
			in:   "go|went|gone",
			want: true,
		}, // pipe separator
		{
			in:   "go.went.gone",
			want: true,
		}, // dot separator
		{
			in:   "go;went,gone",
			want: true,
		}, // mixed non-letter separators
		{
			in:   "went go gone",
			want: false,
		}, // wrong order
		{
			in:   "go gone went",
			want: false,
		}, // wrong order
		{
			in:   "go went",
			want: false,
		}, // missing one
		{
			in:   "go went gone gone",
			want: false,
		}, // extra token
	}
	for _, c := range cases {
		args := checkAllFormsOrderedArgs{
			Input:   c.in,
			Variant: "gb",
		}
		got := s.checkAllFormsOrdered(&goVerb, args)
		require.Equal(t, c.want, got, "checkAllFormsOrdered(%q)", c.in)
	}
}

// TestCheckAllFormsOrderedVariants covers multi-variant forms: any one variant,
// or all of them together, answers a position.
func TestCheckAllFormsOrderedVariants(t *testing.T) {
	s := New(nil, nil)
	be := beVerb() // past gb [was, were], participle [been]
	burn := Verb{  // alternates: any one variant suffices per position
		Base: "burn",
		Past: map[string][]string{
			"gb": {
				"burnt",
				"burned",
			},
		},
		Participle: map[string][]string{
			"gb": {
				"burnt",
				"burned",
			},
		},
	}
	cases := []struct {
		v    Verb
		in   string
		want bool
	}{
		{
			v:    be,
			in:   "be was were been",
			want: true,
		},
		{
			v:    be,
			in:   "be was/were been",
			want: true,
		},
		{
			v:    be,
			in:   "be were was been",
			want: true,
		}, // within past, order-insensitive
		{
			v:    be,
			in:   "be was been",
			want: true,
		}, // one variant of past is enough
		{
			v:    be,
			in:   "be been was were",
			want: false,
		}, // groups out of order
		{
			v:    burn,
			in:   "burn burnt burnt",
			want: true,
		}, // any one variant per position
		{
			v:    burn,
			in:   "burn burned burned",
			want: true,
		}, // the other variant
		{
			v:    burn,
			in:   "burn burnt burned",
			want: true,
		}, // different single variant per position
		{
			v:    burn,
			in:   "burn burned burnt",
			want: true,
		}, // reversed single variants per position
		{
			v:    burn,
			in:   "burn burnt burned burnt",
			want: true,
		}, // may also list both past variants
		{
			v:    burn,
			in:   "burn nope burnt",
			want: false,
		}, // wrong past
	}
	for _, c := range cases {
		args := checkAllFormsOrderedArgs{
			Input:   c.in,
			Variant: "gb",
		}
		got := s.checkAllFormsOrdered(&c.v, args)
		require.Equal(t, c.want, got, "checkAllFormsOrdered(%q)", c.in)
	}
}
