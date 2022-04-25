package twigo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	base_route = "https://api.twitter.com/2/"
)

type Client struct {
	authorizedClient  *http.Client
	consumerKey       string
	consumerSecret    string
	accessToken       string
	accessTokenSecret string
	bearerToken       string
	read_only_access  bool
	userID            string
}

// ** Requests ** //
func (c *Client) request(method, route string, params map[string]interface{}) (*http.Response, error) {
	// OAuth_1a is always true for post and put routes
	dataPayload, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(method, base_route+route, bytes.NewBuffer(dataPayload))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	response, err := c.authorizedClient.Do(request)

	return response, err
}

func (c *Client) get_request(route string, oauth_1a bool, params map[string]interface{}, endpoint_parameters []string) (*http.Response, error) {
	// oauth_1a ==> Whether or not to use OAuth 1.0a User context
	parsedRoute, err := url.Parse(route)
	if err != nil {
		return nil, err
	}

	parameters := url.Values{}
	for param_name, param_value := range params {
		if !contains(endpoint_parameters, param_name) {
			fmt.Printf(" it seems endpoint parameter '%s' is not supported", param_name)
		}
		switch param_valt := param_value.(type) {
		case int:
			parameters.Add(param_name, strconv.Itoa(param_valt))
		case string:
			parameters.Add(param_name, param_valt)
		case []string:
			parameters.Add(param_name, strings.Join(param_valt, ","))
		// TODO: case for arrays of anything else not only string
		// TODO: case datetime
		// 	if param_value.tzinfo is not None:
		// 		param_value = param_value.astimezone(datetime.timezone.utc)
		// 	request_params[param_name] = param_value.strftime("%Y-%m-%dT%H:%M:%S.%fZ")

		default:
			return nil, fmt.Errorf("%s with value of %s is not supported, please contact us", param_name, param_value)
		}

	}
	parsedRoute.RawQuery = parameters.Encode()
	fullRoute := base_route + parsedRoute.String()
	fmt.Println("Route:>> ", fullRoute)
	//%% Conditions below will change oauth kind depend on the specified data in client initialization.
	//%% But they seems wrong.
	if c.read_only_access {
		oauth_1a = false
	}
	if c.bearerToken == "" {
		oauth_1a = true
	}
	if oauth_1a {
		//%% TODO: Should we define authorizedClient here? or tweepy is doing it wrong?
		return c.authorizedClient.Get(fullRoute)
	} else {
		request, err := http.NewRequest("GET", fullRoute, nil)
		if err != nil {
			return nil, err
		}

		request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.bearerToken))
		client := http.Client{}
		return client.Do(request)
	}
}

func (c *Client) delete_request(route string) (*http.Response, error) {
	// OAuth_1a is always true for delete routes
	request, err := http.NewRequest("DELETE", base_route+route, nil)
	if err != nil {
		return nil, err
	}
	return c.authorizedClient.Do(request)
}

// ** Manage Tweets ** //

// Creates a Tweet on behalf of an authenticated user
//
// Parameters
//
// text: Text of the Tweet being created. this field is required if media.media_ids is not present, otherwise pass empty string.
//
// params: A map of parameters.
// you can pass some extra parameters, such as:
// 	"direct_message_deep_link", "for_super_followers_only", "media", "geo", "poll", "reply", "reply_settings", "quote_tweet_id",
// Some of these parameters are a little special and should be passed like this:
// 	media := map[string][]string{
// 		"media_ids": []string{}
// 		"tagged_user_ids": []string{}
// 	}
// 	poll := map[string]interface{}{
// 		"options": map[string]string{},
// 		"duration_minutes": int value,
// 	}
// 	reply := map[string]interface{}{
// 		"in_reply_to_tweet_id": "",
// 		"exclude_reply_user_ids": []string{},
// 	}
// 	geo := map[string]string{"place_id": value}
// 	params := map[string]interface{}{
// 		"text": text
// 		"media": media,
// 		"geo": geo,
// 		"poll": poll,
// 		"reply": reply,
// 	}
//
// Reference
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/manage-tweets/api-reference/post-tweets
func (c *Client) CreateTweet(text string, params map[string]interface{}) (*http.Response, error) {
	if params == nil {
		params = make(map[string]interface{})
	}

	if text != "" {
		params["text"] = text
	} else if params["media"] == nil {
		return nil, fmt.Errorf("text or media is required")
	}

	return c.request(
		"POST",
		"tweets",
		params,
	)
}

// Allows an authenticated user ID to delete a Tweet
//
// Parameters
//
// tweet_id: The Tweet ID you are deleting.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/manage-tweets/api-reference/delete-tweets-id
func (c *Client) DeleteTweet(tweet_id string) (*http.Response, error) {
	route := fmt.Sprintf("tweets/%s", tweet_id)
	return c.delete_request(route)
}

// ** Likes ** //

// Like a Tweet.
//
// Parameters
//
// tweet_id: The ID of the Tweet that you would like to Like.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/likes/api-reference/post-users-id-likesx
func (c *Client) Like(tweet_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"tweet_id": tweet_id,
	}
	route := fmt.Sprintf("users/%s/likes", c.userID)
	return c.request(
		"POST",
		route,
		data,
	)
}

// Unlike a Tweet.
//
// The request succeeds with no action when the user sends a request to a
// user they're not liking the Tweet or have already unliked the Tweet.
//
// Parameters
//
// tweet_id: The ID of the Tweet that you would like to unlike.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/likes/api-reference/delete-users-id-likes-tweet_id
func (c *Client) Unlike(tweet_id string) (*http.Response, error) {
	route := fmt.Sprintf("users/%s/likes/%s", c.userID, tweet_id)
	return c.delete_request(route)
}

// Allows you to get information about a Tweet’s liking users.
//
// Parameters
//
// tweet_id: Tweet ID of the Tweet to request liking users of.
//
// oauth_1a: Whether or not to use OAuth 1.0a. (use false for default)
//
// params (keys):
// 	"expansions", "media.fields", "place.fields",
// 	"poll.fields", "tweet.fields", "user.fields",
// 	"max_results"
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/likes/api-reference/get-tweets-id-liking_users
func (c *Client) GetLikingUsers(tweet_id string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "media.fields", "place.fields",
		"poll.fields", "tweet.fields", "user.fields",
		"max_results", "pagination_token",
	}

	route := fmt.Sprintf("tweets/%s/liking_users", tweet_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

// Allows you to get information about a user’s liked Tweets.
//
// The Tweets returned by this endpoint count towards the Project-level `Tweet cap`.
//
// Parameters
//
// tweet_id: User ID of the user to request liked Tweets for.
//
// oauth_1a: Whether or not to use OAuth 1.0a. (use false for default)
//
// params (keys):
// 	"expansions", "media.fields", "place.fields",
// 	"poll.fields", "tweet.fields", "user.fields",
// 	"max_results"
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/likes/api-reference/get-users-id-liked_tweets
func (c *Client) GetLikedTweets(user_id string, oauth_1a bool, params map[string]interface{}) (*TweetsResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "media.fields",
		"pagination_token", "place.fields", "poll.fields",
		"tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("users/%s/liked_tweets", user_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetsResponse{}).Parse(response)
}

// ** Hide replies ** //

// Hides a reply to a Tweet
//
// Parameters
//
// reply_id:
// 	Unique identifier of the Tweet to hide. The Tweet must belong to a
// 	conversation initiated by the authenticating user.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/hide-replies/api-reference/put-tweets-id-hidden
func (c *Client) HideReply(reply_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"hidden": true,
	}
	route := fmt.Sprintf("tweets/%s/hidden", reply_id)

	return c.request(
		"PUT",
		route,
		data,
	)
}

// Unhides a reply to a Tweet
//
// Parameters
//
// reply_id:
// 	Unique identifier of the Tweet to unhide. The Tweet must belong to
//  a conversation initiated by the authenticating user.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/hide-replies/api-reference/put-tweets-id-hidden
func (c *Client) UnHideReply(reply_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"hidden": false,
	}
	route := fmt.Sprintf("tweets/%s/hidden", reply_id)

	return c.request(
		"PUT",
		route,
		data,
	)
}

// ** Retweets ** //

// Causes the user ID to Retweet the target Tweet.
//
// Parameters
//
// tweet_id: The ID of the Tweet that you would like to Retweet.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/retweets/api-reference/post-users-id-retweets
func (c *Client) Retweet(tweet_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"tweet_id": tweet_id,
	}
	route := fmt.Sprintf("users/%s/retweets", c.userID)
	return c.request(
		"POST",
		route,
		data,
	)
}

// Allows an authenticated user ID to remove the Retweet of a Tweet.
//
// The request succeeds with no action when the user sends a request to a
// user they're not Retweeting the Tweet or have already removed the
// Retweet of.
//
// Parameters
//
// tweet_id: The ID of the Tweet that you would like to remove the Retweet of.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/retweets/api-reference/delete-users-id-retweets-tweet_id
func (c *Client) UnRetweet(tweet_id string) (*http.Response, error) {
	route := fmt.Sprintf("users/%s/retweets/%s", c.userID, tweet_id)
	return c.delete_request(route)
}

// Allows you to get information about who has Retweeted a Tweet.
//
// Parameters
//
// tweet_id: Tweet ID of the Tweet to request Retweeting users of.
//
// oauth_1a: Whether or not to use OAuth 1.0a. (use false for default)
//
// params (keys):
//	"expansions", "media.fields", "place.fields",
// 	"poll.fields", "tweet.fields", "user.fields",
// 	"max_results", "pagination_token"
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/retweets/api-reference/get-tweets-id-retweeted_by
func (c *Client) GetRetweeters(tweet_id string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "media.fields", "place.fields",
		"poll.fields", "tweet.fields", "user.fields",
		"max_results", "pagination_token",
	}
	route := fmt.Sprintf("tweets/%s/retweeted_by", tweet_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

// ** Search tweets ** //

// The full-archive search endpoint returns the complete history of public
// Tweets matching a search query; since the first Tweet was created March
// 26, 2006.
//
// This endpoint is only available to those users who have been approved for the `Academic Research product track`
//
// The Tweets returned by this endpoint count towards the Project-level `Tweet cap`.
//
// Parameters
//
// query : str
// 	One query for matching Tweets. Up to 1024 characters.
// end_time : Union[datetime.datetime, str]
// 	YYYY-MM-DDTHH:mm:ssZ (ISO 8601/RFC 3339). Used with ``start_time``.
// 	The newest, most recent UTC timestamp to which the Tweets will be
// 	provided. Timestamp is in second granularity and is exclusive (for
// 	example, 12:00:01 excludes the first second of the minute). If used
// 	without ``start_time``, Tweets from 30 days before ``end_time``
// 	will be returned by default. If not specified, ``end_time`` will
// 	default to [now - 30 seconds].
// max_results : int
// 	The maximum number of search results to be returned by a request. A
// 	number between 10 and the system limit (currently 500). By default,
// 	a request response will return 10 results.
// next_token : str
// 	This parameter is used to get the next 'page' of results. The value
// 	used with the parameter is pulled directly from the response
// 	provided by the API, and should not be modified. You can learn more
// 	by visiting our page on `pagination`_.
// since_id : Union[int, str]
// 	Returns results with a Tweet ID greater than (for example, more
// 	recent than) the specified ID. The ID specified is exclusive and
// 	responses will not include it. If included with the same request as
// 	a ``start_time`` parameter, only ``since_id`` will be used.
// start_time : Union[datetime.datetime, str]
// 	YYYY-MM-DDTHH:mm:ssZ (ISO 8601/RFC 3339). The oldest UTC timestamp
// 	from which the Tweets will be provided. Timestamp is in second
// 	granularity and is inclusive (for example, 12:00:01 includes the
// 	first second of the minute). By default, a request will return
// 	Tweets from up to 30 days ago if you do not include this parameter.
// until_id: string
// 	Returns results with a Tweet ID less than (that is, older than) the
// 	specified ID. Used with ``since_id``. The ID specified is exclusive
// 	and responses will not include it.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/search/api-reference/get-tweets-search-all
//
// Academic Research product track: https://developer.twitter.com/en/docs/projects/overview#product-track
//
// Tweet cap: https://developer.twitter.com/en/docs/projects/overview#tweet-cap
//
// pagination: https://developer.twitter.com/en/docs/twitter-api/tweets/search/integrate/paginate
func (c *Client) SearchAllTweets(query string, params map[string]interface{}) (*TweetsResponse, error) {
	endpoint_parameters := []string{
		"end_time", "expansions", "max_results", "media.fields",
		"next_token", "place.fields", "poll.fields", "query",
		"since_id", "start_time", "tweet.fields", "until_id",
		"user.fields",
	}
	route := "tweets/search/all"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["query"] = query
	response, err := c.get_request(route, false, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetsResponse{}).Parse(response)
}

// The recent search endpoint returns Tweets from the last seven days that match a search query.
//
// The Tweets returned by this endpoint count towards the Project-level
// `Tweet cap`.
//
// Parameters
//
// query : str
// 	One rule for matching Tweets. If you are using a
// 	`Standard Project`_ at the Basic `access level`_, you can use the
// 	basic set of `operators`_ and can make queries up to 512 characters
// 	long. If you are using an `Academic Research Project`_ at the Basic
// 	access level, you can use all available operators and can make
// 	queries up to 1,024 characters long.
// end_time : Union[datetime.datetime, str]
// 	YYYY-MM-DDTHH:mm:ssZ (ISO 8601/RFC 3339). The newest, most recent
// 	UTC timestamp to which the Tweets will be provided. Timestamp is in
// 	second granularity and is exclusive (for example, 12:00:01 excludes
// 	the first second of the minute). By default, a request will return
// 	Tweets from as recent as 30 seconds ago if you do not include this
// 	parameter.
// max_results : int
// 	The maximum number of search results to be returned by a request. A
// 	number between 10 and 100. By default, a request response will
// 	return 10 results.
// next_token : str
// 	This parameter is used to get the next 'page' of results. The value
// 	used with the parameter is pulled directly from the response
// 	provided by the API, and should not be modified.
// since_id : Union[int, str]
// 	Returns results with a Tweet ID greater than (that is, more recent
// 	than) the specified ID. The ID specified is exclusive and responses
// 	will not include it. If included with the same request as a
// 	``start_time`` parameter, only ``since_id`` will be used.
// start_time : Union[datetime.datetime, str]
// 	YYYY-MM-DDTHH:mm:ssZ (ISO 8601/RFC 3339). The oldest UTC timestamp
// 	(from most recent seven days) from which the Tweets will be
// 	provided. Timestamp is in second granularity and is inclusive (for
// 	example, 12:00:01 includes the first second of the minute). If
// 	included with the same request as a ``since_id`` parameter, only
// 	``since_id`` will be used. By default, a request will return Tweets
// 	from up to seven days ago if you do not include this parameter.
// until_id : Union[int, str]
// 	Returns results with a Tweet ID less than (that is, older than) the
// 	specified ID. The ID specified is exclusive and responses will not
// 	include it.
//
// References
//
// https://developer.twitter.com/en/docs/twitter-api/tweets/search/api-reference/get-tweets-search-recent
//
// Tweet cap: https://developer.twitter.com/en/docs/projects/overview#tweet-cap
//
// Standard Project: https://developer.twitter.com/en/docs/projects
//
// access level: https://developer.twitter.com/en/products/twitter-api/early-access/guide.html#na_1
//
// operators: https://developer.twitter.com/en/docs/twitter-api/tweets/search/integrate/build-a-query
//
// Academic Research Project: https://developer.twitter.com/en/docs/projects
func (c *Client) SearchRecentTweets(query string, oauth_1a bool, params map[string]interface{}) (*TweetsResponse, error) {
	endpoint_parameters := []string{
		"end_time", "expansions", "max_results", "media.fields",
		"next_token", "place.fields", "poll.fields", "query",
		"since_id", "start_time", "tweet.fields", "until_id",
		"user.fields",
	}
	route := "tweets/search/recent"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["query"] = query
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetsResponse{}).Parse(response)
}

// ** Timelines ** //
func (c *Client) GetUserTweets(user_id string, oauth_1a bool, params map[string]interface{}) (*TweetsResponse, error) {
	endpoint_parameters := []string{
		"end_time", "exclude", "expansions", "max_results",
		"media.fields", "pagination_token", "place.fields",
		"poll.fields", "since_id", "start_time", "tweet.fields",
		"until_id", "user.fields",
	}
	route := fmt.Sprintf("users/%s/tweets", user_id)

	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetsResponse{}).Parse(response)
}

func (c *Client) GetUserMentions(user_id string, oauth_1a bool, params map[string]interface{}) (*TweetsResponse, error) {
	endpoint_parameters := []string{
		"end_time", "expansions", "max_results", "media.fields",
		"pagination_token", "place.fields", "poll.fields", "since_id",
		"start_time", "tweet.fields", "until_id", "user.fields",
	}

	route := fmt.Sprintf("users/%s/mentions", user_id)

	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetsResponse{}).Parse(response)
}

// ** Tweet counts ** //
func (c *Client) GetAllTweetsCount(query string, params map[string]interface{}) (*http.Response, error) {
	endpoint_parameters := []string{
		"end_time", "granularity", "next_token", "query",
		"since_id", "start_time", "until_id",
	}
	if params == nil {
		params = make(map[string]interface{})
	}
	params["query"] = query
	return c.get_request("tweets/counts/all", false, params, endpoint_parameters)
}

func (c *Client) GetRecentTweetsCount(query string, params map[string]interface{}) (*http.Response, error) {
	endpoint_parameters := []string{
		"end_time", "granularity", "query",
		"since_id", "start_time", "until_id",
	}
	if params == nil {
		params = make(map[string]interface{})
	}
	params["query"] = query
	return c.get_request("tweets/counts/recent", false, params, endpoint_parameters)
}

// ** Tweet lookup ** //
func (c *Client) GetTweet(tweet_id string, oauth_1a bool, params map[string]interface{}) (*TweetResponse, error) {
	endpoint_parameters := []string{
		"expansions", "media.fields", "place.fields",
		"poll.fields", "tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("tweets/%s", tweet_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetResponse{}).Parse(response)
}

func (c *Client) GetTweets(tweet_ids []string, oauth_1a bool, params map[string]interface{}) (*TweetsResponse, error) {
	endpoint_parameters := []string{
		"ids", "expansions", "media.fields", "place.fields",
		"poll.fields", "tweet.fields", "user.fields",
	}
	if params == nil {
		params = make(map[string]interface{})
	}
	params["ids"] = tweet_ids
	response, err := c.get_request("tweets", oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetsResponse{}).Parse(response)
}

// ** Blocks ** //
func (c *Client) Block(target_user_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"target_user_id": target_user_id,
	}
	route := fmt.Sprintf("users/%s/blocking", c.userID)

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) UnBlock(target_user_id string) (*http.Response, error) {
	route := fmt.Sprintf("users/%s/blocking/%s", c.userID, target_user_id)
	return c.delete_request(route)
}

func (c *Client) GetBlocked(params map[string]interface{}) (*http.Response, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "tweet.fields",
		"user.fields", "pagination_token",
	}
	route := fmt.Sprintf("users/%s/blocking", c.userID)
	return c.get_request(route, true, params, endpoint_parameters)
}

// ** Follows ** //
func (c *Client) FollowUser(target_user_id string, params map[string]interface{}) (*http.Response, error) {
	data := map[string]interface{}{
		"target_user_id": target_user_id,
	}

	route := fmt.Sprintf("users/%s/following", c.userID)

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) UnfollowUser(target_user_id string) (*http.Response, error) {
	route := fmt.Sprintf("users/%s/following/%s", c.userID, target_user_id)
	return c.delete_request(route)
}

func (c *Client) GetUserFollowers(user_id string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "tweet.fields",
		"user.fields", "pagination_token",
	}
	route := fmt.Sprintf("users/%s/followers", user_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

func (c *Client) GetUserFollowing(user_id string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "tweet.fields",
		"user.fields", "pagination_token",
	}
	route := fmt.Sprintf("users/%s/following", user_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

// ** Mutes ** //
func (c *Client) Mute(target_user_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"target_user_id": target_user_id,
	}

	route := fmt.Sprintf("users/%s/muting", c.userID)

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) UnMute(target_user_id string) (*http.Response, error) {
	route := fmt.Sprintf("users/%s/muting/%s", c.userID, target_user_id)
	return c.delete_request(route)
}

func (c *Client) GetMuted(oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "tweet.fields",
		"user.fields", "pagination_token",
	}
	route := fmt.Sprintf("users/%s/muting", c.userID)
	response, err := c.get_request(route, true, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

// ** User lookup ** //
func (c *Client) GetUserByID(user_id string, oauth_1a bool, params map[string]interface{}) (*UserResponse, error) {
	endpoint_parameters := []string{
		"expansions", "tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("users/%s", user_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UserResponse{}).Parse(response)
}

func (c *Client) GetUserByUsername(username string, oauth_1a bool, params map[string]interface{}) (*UserResponse, error) {
	endpoint_parameters := []string{
		"expansions", "tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("users/by/username/%s", username)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UserResponse{}).Parse(response)
}

func (c *Client) GetUsersByIDs(user_ids []string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"usernames", "ids", "expansions",
		"tweet.fields", "user.fields",
	}

	if user_ids == nil {
		return nil, fmt.Errorf("user_ids are required")
	}
	if params == nil {
		params = make(map[string]interface{})
	}
	params["ids"] = user_ids

	response, err := c.get_request("users", oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

func (c *Client) GetUsersByUsernames(usernames []string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"usernames", "ids", "expansions",
		"tweet.fields", "user.fields",
	}

	if usernames == nil {
		return nil, fmt.Errorf("usernames are required")
	}
	if params == nil {
		params = make(map[string]interface{})
	}
	params["usernames"] = usernames

	response, err := c.get_request("users/by", oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

// ** Spaces ** //
func (c *Client) SearchSpaces(query string, params map[string]interface{}) (*SpacesResponse, error) {
	endpoint_parameters := []string{
		"query", "expansions", "max_results",
		"space.fields", "state", "user.fields",
	}
	route := "spaces/search"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["query"] = query
	response, err := c.get_request(route, false, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&SpacesResponse{}).Parse(response)
}

func (c *Client) GetSpacesBySpaceIDs(space_ids []string, params map[string]interface{}) (*SpacesResponse, error) {
	endpoint_parameters := []string{
		"ids", "user_ids", "expansions", "space.fields", "user.fields",
	}
	route := "spaces"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["ids"] = space_ids
	response, err := c.get_request(route, false, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&SpacesResponse{}).Parse(response)
}

func (c *Client) GetSpacesByCreatorIDs(creator_ids []string, params map[string]interface{}) (*SpacesResponse, error) {
	endpoint_parameters := []string{
		"ids", "user_ids", "expansions", "space.fields", "user.fields",
	}
	route := "spaces/by/creator_ids"
	if params == nil {
		params = make(map[string]interface{})
	}
	params["user_ids"] = creator_ids
	response, err := c.get_request(route, false, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&SpacesResponse{}).Parse(response)
}

func (c *Client) GetSpace(space_id string, params map[string]interface{}) (*SpaceResponse, error) {
	endpoint_parameters := []string{
		"expansions", "space.fields", "user.fields",
	}
	route := fmt.Sprintf("spaces/%s", space_id)
	response, err := c.get_request(route, false, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&SpaceResponse{}).Parse(response)
}

func (c *Client) GetSpaceBuyers(space_id string, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "media.fields", "place.fields",
		"poll.fields", "tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("spaces/%s/buyers", space_id)
	response, err := c.get_request(route, false, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

// ** List Tweets lookup ** //
func (c *Client) GetListTweets(list_id string, oauth_1a bool, params map[string]interface{}) (*TweetsResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "pagination_token",
		"tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("lists/%s/tweets", list_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&TweetsResponse{}).Parse(response)
}

// ** List follows ** //
func (c *Client) FollowList(list_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"list_id": list_id,
	}

	route := fmt.Sprintf("users/%s/followed_lists", c.userID)

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) UnfollowList(list_id string) (*http.Response, error) {
	route := fmt.Sprintf("users/%s/followed_lists/%s", c.userID, list_id)
	return c.delete_request(route)
}

func (c *Client) GetListFollowers(list_id string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "pagination_token",
		"tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("lists/%s/followers", list_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

func (c *Client) GetFollowedLists(user_id string, oauth_1a bool, params map[string]interface{}) (*ListsResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "pagination_token",
		"list.fields", "user.fields",
	}
	route := fmt.Sprintf("users/%s/followed_lists", user_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&ListsResponse{}).Parse(response)
}

// ** List lookup ** //
func (c *Client) GetList(list_id string, oauth_1a bool, params map[string]interface{}) (*ListResponse, error) {
	endpoint_parameters := []string{
		"expansions", "list.fields", "user.fields",
	}
	route := fmt.Sprintf("lists/%s", list_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&ListResponse{}).Parse(response)
}

func (c *Client) GetOwnedLists(user_id string, oauth_1a bool, params map[string]interface{}) (*ListsResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "pagination_token",
		"list.fields", "user.fields",
	}
	route := fmt.Sprintf("users/%s/owned_lists", user_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&ListsResponse{}).Parse(response)
}

// ** List members ** //
func (c *Client) AddListMemeber(list_id, user_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"user_id": user_id,
	}

	route := fmt.Sprintf("lists/%s/members", list_id)

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) RemoveListMember(list_id, user_id string) (*http.Response, error) {
	route := fmt.Sprintf("lists/%s/members/%s", list_id, user_id)
	return c.delete_request(route)
}

func (c *Client) GetListMembers(list_id string, oauth_1a bool, params map[string]interface{}) (*UsersResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "pagination_token",
		"tweet.fields", "user.fields",
	}
	route := fmt.Sprintf("lists/%s/members", list_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&UsersResponse{}).Parse(response)
}

func (c *Client) GetListMemberships(user_id string, oauth_1a bool, params map[string]interface{}) (*ListsResponse, error) {
	endpoint_parameters := []string{
		"expansions", "max_results", "pagination_token",
		"list.fields", "user.fields",
	}
	route := fmt.Sprintf("users/%s/list_memberships", user_id)
	response, err := c.get_request(route, oauth_1a, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&ListsResponse{}).Parse(response)
}

// ** Manage Lists ** //
func (c *Client) CreateList(name string, description string, private bool, params map[string]interface{}) (*http.Response, error) {
	data := map[string]interface{}{
		"name":        name,
		"description": description,
		"private":     private,
	}

	route := "lists"

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) UpdateList(list_id string, name string, description string, private bool, params map[string]interface{}) (*http.Response, error) {
	data := map[string]interface{}{
		"name":        name,
		"description": description,
		"private":     private,
	}

	route := fmt.Sprintf("lists/%s", list_id)

	return c.request(
		"PUT",
		route,
		data,
	)
}

func (c *Client) DeleteList(list_id string) (*http.Response, error) {
	route := fmt.Sprintf("lists/%s", list_id)
	return c.delete_request(route)
}

// ** Pinned Lists ** //
func (c *Client) PinList(list_id string) (*http.Response, error) {
	data := map[string]interface{}{
		"list_id": list_id,
	}

	route := fmt.Sprintf("users/%s/pinned_lists", c.userID)

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) UnpinList(list_id string) (*http.Response, error) {
	route := fmt.Sprintf("users/%s/pinned_lists/%s", c.userID, list_id)
	return c.delete_request(route)
}

func (c *Client) GetPinnedLists(params map[string]interface{}) (*ListsResponse, error) {
	endpoint_parameters := []string{
		"expansions", "list.fields", "user.fields",
	}
	route := fmt.Sprintf("users/%s/pinned_lists", c.userID)
	response, err := c.get_request(route, true, params, endpoint_parameters)
	if err != nil {
		return nil, err
	}
	return (&ListsResponse{}).Parse(response)
}

// ** Batch Compliance ** //
func (c *Client) CreateComplianceJobs(job_type, name, resumable string) (*http.Response, error) {
	data := map[string]interface{}{
		"type": job_type,
	}
	if name != "" {
		data["name"] = name
	}
	if resumable != "" {
		data["resumable"] = resumable
	}

	route := "compliance/jobs"

	return c.request(
		"POST",
		route,
		data,
	)
}

func (c *Client) GetComplianceJob(job_id string) (*http.Response, error) {
	route := fmt.Sprintf("compliance/jobs/%s", job_id)
	return c.get_request(route, false, nil, nil)
}

func (c *Client) GetComplianceJobs(job_type string, params map[string]interface{}) (*http.Response, error) {
	endpoint_parameters := []string{
		"type", "status",
	}
	if params == nil {
		params = map[string]interface{}{}
	}
	params["type"] = job_type

	route := "compliance/jobs"
	return c.get_request(route, false, params, endpoint_parameters)
}

// func QueryMaker() string
// func (c *Client) GetMe() *Response
