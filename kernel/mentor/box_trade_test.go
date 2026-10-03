package mentor

import (
	"testing"
)

// R2 [00-METHOD Risk-reward, D3.4 p3 @ 07:38]: confluence = box edge + key
// level inside the box or within 2 pts of its edge + the 5m trigger agrees.
// LONG = FTGL (support); SHORT = FTGH. Feeds DS-102's exit-C / size-10.
func TestConfluenceVerdict(t *testing.T) {
	b := Box{Kind: FTGL, Top: 29020, Bottom: 29000}    // support box
	kls := []Level{{Kind: KindKeyLevel, Price: 28998}} // 2 pts below the edge → "at"
	buy := TriggerLine{Dir: SideLong, Price: 28990}    // buy line under the entry → agrees
	sell := TriggerLine{Dir: SideShort, Price: 29100}

	if f := ConfluenceVerdict(b, SideLong, kls, buy); !f.On || f.Side != SideLong {
		t.Fatalf("FTGL + key level 2 pts off the edge + buy trigger = %+v, want On Long", f)
	}
	if f := ConfluenceVerdict(b, SideLong, kls, sell); f.On {
		t.Fatal("sell trigger must not confluence a long")
	}
	if f := ConfluenceVerdict(b, SideLong, kls, TriggerLine{}); f.On {
		t.Fatal("no trigger line can never agree — fail closed")
	}
	far := []Level{{Kind: KindKeyLevel, Price: 28950}}
	if f := ConfluenceVerdict(b, SideLong, far, buy); f.On {
		t.Fatal("key level 50 pts away is not 'at' the box")
	}
	inside := []Level{{Kind: KindKeyLevel, Price: 29010}}
	if f := ConfluenceVerdict(b, SideLong, inside, buy); !f.On {
		t.Fatal("a key level INSIDE the box must confluence")
	}
	// SHORT mirror: FTGH + key level + sell trigger.
	c := Box{Kind: FTGH, Top: 29100, Bottom: 29080}
	skl := []Level{{Kind: KindKeyLevel, Price: 29102}} // 2 pts above the edge
	if f := ConfluenceVerdict(c, SideShort, skl, sell); !f.On || f.Side != SideShort {
		t.Fatalf("FTGH + key level 2 pts off the edge + sell trigger = %+v, want On Short", f)
	}
	if f := ConfluenceVerdict(c, SideLong, skl, buy); f.On {
		t.Fatal("a long against an FTGH must not confluence")
	}
	// Wrong box kind for the side.
	if f := ConfluenceVerdict(b, SideShort, skl, sell); f.On {
		t.Fatal("SHORT needs an FTGH, not an FTGL")
	}
}
