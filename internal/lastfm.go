package lib

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type LastFMTrack struct {
	XMLName    xml.Name `xml:"track"`
	NowPlaying string   `xml:"nowplaying,attr"`
	Artist     struct {
		Name string `xml:",chardata"`
		MBID string `xml:"mbid,attr"`
	} `xml:"artist"`
	Name  string `xml:"name"`
	Album struct {
		Name string `xml:",chardata"`
		MBID string `xml:"mbid,attr"`
	} `xml:"album"`
	URL  string `xml:"url"`
	Date struct {
		Text string `xml:",chardata"`
		UTS  string `xml:"uts,attr"`
	} `xml:"date"`
	Images []struct {
		Size string `xml:"size,attr"`
		URL  string `xml:",chardata"`
	} `xml:"image"`
	Streamable string `xml:"streamable"`
	MBID       string `xml:"mbid"`
}

type LastFMRecentTracks struct {
	XMLName    xml.Name      `xml:"recenttracks"`
	User       string        `xml:"user,attr"`
	Page       string        `xml:"page,attr"`
	PerPage    string        `xml:"perPage,attr"`
	TotalPages string        `xml:"totalPages,attr"`
	Total      string        `xml:"total,attr"`
	Tracks     []LastFMTrack `xml:"track"`
}

type LastFMResponse struct {
	XMLName      xml.Name           `xml:"lfm"`
	Status       string             `xml:"status,attr"`
	RecentTracks LastFMRecentTracks `xml:"recenttracks"`
}

type FriendlyTrack struct {
	Title              string `json:"title"`
	Artist             string `json:"artist"`
	Album              string `json:"album"`
	EpochTimePlayed    int64  `json:"epoch_time_played"`
	ArtworkURL         string `json:"artwork_url"`
	IsCurrentlyPlaying bool   `json:"is_currently_playing"`
}

type FriendlyRecentTracks struct {
	Tracks      []FriendlyTrack `json:"tracks"`
	TotalTracks int             `json:"total_tracks"`
	Cached      bool
}

func ConvertToFriendlyTrack(track LastFMTrack) FriendlyTrack {
	utsTime, _ := strconv.ParseInt(track.Date.UTS, 10, 64)

	sizes := []string{"extralarge", "large", "medium", "small"}
	artworkURL := ""
	for _, size := range sizes {
		for _, img := range track.Images {
			if img.Size == size {
				artworkURL = img.URL
				break
			}
		}
		if artworkURL != "" {
			break
		}
	}

	return FriendlyTrack{
		Title:              track.Name,
		Artist:             track.Artist.Name,
		Album:              track.Album.Name,
		EpochTimePlayed:    utsTime,
		ArtworkURL:         artworkURL,
		IsCurrentlyPlaying: track.NowPlaying == "true",
	}
}

func fetchLastFMRecentTracksXML(apiKey, username string, limit int) (*LastFMResponse, error) {
	baseURL := "https://ws.audioscrobbler.com/2.0/"

	params := url.Values{}
	params.Add("method", "user.getrecenttracks")
	params.Add("user", username)
	params.Add("api_key", apiKey)
	params.Add("limit", strconv.Itoa(limit))

	fullURL := baseURL + "?" + params.Encode()

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
	}

	var lastfmResponse LastFMResponse
	err = xml.NewDecoder(resp.Body).Decode(&lastfmResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing XML: %v", err)
	}

	return &lastfmResponse, nil
}

func GetRecentTracksWithFriendlyFormat(apiKey, username string, limit int) (FriendlyRecentTracks, error) {
	xmlResponse, err := fetchLastFMRecentTracksXML(apiKey, username, limit)
	if err != nil {
		return FriendlyRecentTracks{}, err
	}

	friendlyTracks := make([]FriendlyTrack, 0, len(xmlResponse.RecentTracks.Tracks))
	for _, track := range xmlResponse.RecentTracks.Tracks {
		friendlyTracks = append(friendlyTracks, ConvertToFriendlyTrack(track))
	}

	return FriendlyRecentTracks{
		Tracks:      friendlyTracks,
		TotalTracks: len(friendlyTracks),
	}, nil
}
